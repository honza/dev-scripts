import React, { Component } from "react";
import { FaSpinner } from "react-icons/fa";

let buildResultClassNames = (build) =>
  build.passed
    ? `bg-green-400 hover:bg-green-300 border-green-600`
    : `bg-red-300 hover:bg-red-300 border-red-600`;

class Version extends Component {
  render() {
    let version = this.props.version;
    return (
      <div className="p-8 bg-white rounded-lg shadow-lg font-mono">
        <h1 className="text-center font-bold text-4xl text-slate-600 mb-10">
          {version.name}
        </h1>

        <div className="text-sm grid grid-cols-2 gap-2">
          {version.builds.map((build) => (
            <div
              className={`p-1 px-3 border-2 rounded-lg ${buildResultClassNames(
                build
              )}`}
            >
              <a href={build.url} target="_blank" rel="noreferrer">
                {build.in_progress ? <FaSpinner className="inline mr-2" /> : []}
                {build.job_name}
              </a>
            </div>
          ))}
        </div>
      </div>
    );
  }
}

export default Version;
