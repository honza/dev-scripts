import React, { useRef } from "react";
import { FaSpinner, FaCode } from "react-icons/fa";
import { SiEquinixmetal } from "react-icons/si";
import { BiError } from "react-icons/bi";
import { BsFileFont } from "react-icons/bs";

import Tooltips from "@material-tailwind/react/Tooltips";
import TooltipsContent from "@material-tailwind/react/TooltipsContent";
import TimeAgo from "react-timeago";

let bg = (build) => (build.new_build_in_progress ? "100" : "400");

let buildResultClassNames = (build) =>
  build.passed
    ? `bg-green-${bg(build)} hover:bg-green-300 border-green-600`
    : `bg-red-${bg(build)} hover:bg-red-300 border-red-600`;

let iconClasses = "inline mr-2 mt-1 absolute right-0";

let failureIcon = (build) => {
  switch (build.failure_reason) {
    case "devscripts-setup":
      return <FaCode className={iconClasses} />;
    case "e2e-test":
      return <BsFileFont className={iconClasses} />;
    case "packet-setup":
      return <SiEquinixmetal className={iconClasses} />;
    case "unknown":
      return <BiError className={iconClasses} />;
    default:
      return [];
  }
};

const Build = ({ build }) => {
  const ref = useRef();

  return (
    <>
      <div
        key={build.build_id}
        ref={ref}
        className={`p-1 px-3 border-2 rounded-lg relative ${buildResultClassNames(
          build
        )}`}
      >
        <a href={build.url} target="_blank" rel="noreferrer">
          {build.new_build_in_progress ? (
            <FaSpinner className="inline mr-2 animate-spin" />
          ) : (
            []
          )}
          {build.job_name}
        </a>
        {failureIcon(build)}
      </div>

      <Tooltips placement="left" ref={ref}>
        <TooltipsContent>
          Job finished: <TimeAgo date={build.finished} />
        </TooltipsContent>
      </Tooltips>
    </>
  );
};

export default Build;
