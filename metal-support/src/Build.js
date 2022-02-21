import React, { Component } from "react";
import { FaSpinner } from "react-icons/fa";

let buildResultClassNames = (build) =>
  build.passed
    ? `bg-green-400 hover:bg-green-300 border-green-600`
    : `bg-red-300 hover:bg-red-300 border-red-600`;

export default class Build extends Component {
  render() {
    let build = this.props.build;
    return (
      <div
        key={build.build_id}
        className={`p-1 px-3 border-2 rounded-lg ${buildResultClassNames(
          build
        )}`}
      >
        <a href={build.url} target="_blank" rel="noreferrer">
          {build.in_progress ? <FaSpinner className="inline mr-2" /> : []}
          {build.job_name}
        </a>
      </div>
    );
  }
}
