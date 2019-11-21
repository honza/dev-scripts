#!/bin/bash

set -ex

source logging.sh
source common.sh
source rhcos.sh
source ocp_install_env.sh

# To replace an image entry in the openshift releae image, set
# <ENTRYNAME>_LOCAL_IMAGE - where ENTRYNAME matches an uppercase version of the name in the release image
# with "-" converted to "_" e.g. to use a custom ironic-inspector
#export IRONIC_INSPECTOR_LOCAL_IMAGE=https://github.com/metal3-io/ironic-inspector-image
#export IRONIC_RHCOS_DOWNLOADER_LOCAL_IMAGE=https://github.com/openshift-metal3/rhcos-downloader
#export BAREMETAL_OPERATOR_LOCAL_IMAGE=192.168.111.1:5000/localimages/bmo:latest
rm -f assets/templates/99_local-registry.yaml $OPENSHIFT_INSTALL_PATH/data/data/bootstrap/baremetal/files/etc/containers/registries.conf

# Various commands here need the Pull Secret in a file
export REGISTRY_AUTH_FILE=$(mktemp "pullsecret--XXXXXXXXXX")
{ echo "${PULL_SECRET}" ; } 2> /dev/null > $REGISTRY_AUTH_FILE

DOCKERFILE=$(mktemp "release-update--XXXXXXXXXX")
echo "FROM $OPENSHIFT_RELEASE_IMAGE" > $DOCKERFILE
for IMAGE_VAR in $(env | grep "_LOCAL_IMAGE=" | grep -o "^[^=]*") ; do
    IMAGE=${!IMAGE_VAR}

    sudo -E podman pull $OPENSHIFT_RELEASE_IMAGE

    # Is it a git repo?
    if [[ "$IMAGE" =~ "://" ]] ; then
        REPOPATH=~/${IMAGE##*/}
        # Clone to ~ if not there already
        [ -e "$REPOPATH" ] || git clone $IMAGE $REPOPATH
        cd $REPOPATH
        export $IMAGE_VAR=${IMAGE##*/}:latest
        export $IMAGE_VAR=$LOCAL_REGISTRY_ADDRESS/localimages/${!IMAGE_VAR}
        sudo podman build -t ${!IMAGE_VAR} .
        cd -
        sudo podman push --tls-verify=false ${!IMAGE_VAR} ${!IMAGE_VAR}
    fi

    # Update the bootstrap and master nodes to treat LOCAL_REGISTRY_ADDRESS as insecure
    mkdir -p $OPENSHIFT_INSTALL_PATH/data/data/bootstrap/baremetal/files/etc/containers
    echo -e "[registries.insecure]\nregistries = ['${LOCAL_REGISTRY_ADDRESS}']" > $OPENSHIFT_INSTALL_PATH/data/data/bootstrap/baremetal/files/etc/containers/registries.conf
    cp assets/templates/99_local-registry.yaml.optional assets/templates/99_local-registry.yaml

    IMAGE_NAME=$(echo ${IMAGE_VAR/_LOCAL_IMAGE} | tr '[:upper:]_' '[:lower:]-')
    OLDIMAGE=$(sudo podman run --rm $OPENSHIFT_RELEASE_IMAGE image $IMAGE_NAME)
    echo "RUN sed -i 's%$OLDIMAGE%${!IMAGE_VAR}%g' /release-manifests/*" >> $DOCKERFILE
done

if [ ! -z "${MIRROR_IMAGES}" ]; then
    oc adm release mirror \
       --insecure=true \
        -a $REGISTRY_AUTH_FILE  \
        --from $OPENSHIFT_RELEASE_IMAGE \
        --to $LOCAL_REGISTRY_ADDRESS/localimages/local-release-image
fi

if [ -f assets/templates/99_local-registry.yaml ] ; then
    build_installer
    sudo podman image build -t $OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE -f $DOCKERFILE
    sudo podman push --tls-verify=false $OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE $OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE
fi
rm -f $DOCKERFILE

for name in ironic ironic-api ironic-conductor ironic-inspector dnsmasq httpd mariadb ipa-downloader coreos-downloader vbmc sushy-tools; do
    sudo podman ps | grep -w "$name$" && sudo podman kill $name
    sudo podman ps --all | grep -w "$name$" && sudo podman rm $name -f
done

# Remove existing pod
if  sudo podman pod exists ironic-pod ; then 
    sudo podman pod rm ironic-pod -f
fi

# Create pod
sudo podman pod create -n ironic-pod 

# Pull the rhcos-downloder image to use from the release, this gets change
# to use IRONIC_RHCOS_DOWNLOADER_LOCAL_IMAGE if present
IRONIC_RHCOS_DOWNLOADER_IMAGE=$(oc adm release info --registry-config $REGISTRY_AUTH_FILE $OPENSHIFT_RELEASE_IMAGE --image-for=ironic-rhcos-downloader)

IRONIC_IMAGE=${IRONIC_LOCAL_IMAGE:-$IRONIC_IMAGE}
IRONIC_IPA_DOWNLOADER_IMAGE=${IRONIC_IPA_DOWNLOADER_LOCAL_IMAGE:-$IRONIC_IPA_DOWNLOADER_IMAGE}
IRONIC_RHCOS_DOWNLOADER_IMAGE=${IRONIC_RHCOS_DOWNLOADER_LOCAL_IMAGE:-$IRONIC_RHCOS_DOWNLOADER_IMAGE}

for IMAGE in ${IRONIC_IMAGE} ${IRONIC_IPA_DOWNLOADER_IMAGE} ${IRONIC_RHCOS_DOWNLOADER_IMAGE} ${VBMC_IMAGE} ${SUSHY_TOOLS_IMAGE} ; do
    sudo -E podman pull $([[ $IMAGE =~ $LOCAL_REGISTRY_ADDRESS.* ]] && echo "--tls-verify=false" ) $IMAGE
done

rm -rf $REGISTRY_AUTH_FILE

# cached images to the bootstrap VM
sudo podman run -d --net host --privileged --name httpd --pod ironic-pod \
     -v $IRONIC_DATA_DIR:/shared --entrypoint /bin/runhttpd ${IRONIC_IMAGE}

sudo podman run -d --net host --privileged --name ipa-downloader --pod ironic-pod \
     -v $IRONIC_DATA_DIR:/shared ${IRONIC_IPA_DOWNLOADER_IMAGE} /usr/local/bin/get-resource.sh

sudo podman run -d --net host --privileged --name coreos-downloader --pod ironic-pod \
     -v $IRONIC_DATA_DIR:/shared ${IRONIC_RHCOS_DOWNLOADER_IMAGE} /usr/local/bin/get-resource.sh $RHCOS_IMAGE_URL

if [ "$NODES_PLATFORM" = "libvirt" ]; then
    sudo podman run -d --net host --privileged --name vbmc --pod ironic-pod \
         -v "$WORKING_DIR/virtualbmc/vbmc":/root/.vbmc -v "/root/.ssh":/root/ssh \
         "${VBMC_IMAGE}"
    
    sudo podman run -d --net host --privileged --name sushy-tools --pod ironic-pod \
         -v "$WORKING_DIR/virtualbmc/sushy-tools":/root/sushy -v "/root/.ssh":/root/ssh \
         "${SUSHY_TOOLS_IMAGE}"
fi


# Wait for the downloader containers to finish, if they are updating an existing cache
# the checks below will pass because old data exists
sudo podman wait -i 1000 ipa-downloader coreos-downloader

# Wait for images to be downloaded/ready
while ! curl --fail http://localhost/images/rhcos-ootpa-latest.qcow2.md5sum ; do sleep 1 ; done
while ! curl --fail --head http://localhost/images/ironic-python-agent.initramfs ; do sleep 1; done
while ! curl --fail --head http://localhost/images/ironic-python-agent.tar.headers ; do sleep 1; done
while ! curl --fail --head http://localhost/images/ironic-python-agent.kernel ; do sleep 1; done
