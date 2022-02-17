This folder is meant to contain all the scripts / utilities required to help
the activity of the metal-support role, in order to efficiently and quickly
monitor the health of the OpenShift CI release jobs for the metal platform

Frontend
--------

The frontend uses [tailwindcss](https://tailwindcss.com/).

You will need to have a recent version of [node.js](https://nodejs.org/en/)
installed.  If you use Nix, you can do use the provided `shell.nix` file.

Then, install dependencies:

```
$ npm install
```

Then, start the tailwindcss watch process with:

```
$ make css
```

Now you can change the `templates/index.html` file, and the `css/style.min.css`
will be updated accordingly.

Any changes to `css/style.min.css` should be committed along with changes to
`templates/index.html`.
