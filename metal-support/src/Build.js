import React, { useRef } from "react";
import { FaSpinner } from "react-icons/fa";
import Tooltips from "@material-tailwind/react/Tooltips";
import TooltipsContent from "@material-tailwind/react/TooltipsContent";
import TimeAgo from "react-timeago";

let buildResultClassNames = (build) =>
  build.passed
    ? `bg-green-400 hover:bg-green-300 border-green-600`
    : `bg-red-400 hover:bg-red-300 border-red-600`;

const Build = ({ build }) => {
  const ref = useRef();

  return (
    <>
      <div
        key={build.build_id}
        ref={ref}
        className={`p-1 px-3 border-2 rounded-lg ${buildResultClassNames(
          build
        )}`}
      >
        <a href={build.url} target="_blank" rel="noreferrer">
          {build.new_build_in_progress ? (
            <FaSpinner className="inline mr-2" />
          ) : (
            []
          )}
          {build.job_name}
        </a>
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
