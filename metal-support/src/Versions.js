import React, { Component } from "react";
import Version from "./Version";

class Versions extends Component {
  render() {
    console.log(this.props.versions);
    return this.props.loaded ? (
      <div className="flex flex-row items-stretch gap-4 w-full p-3">
        {this.props.versions.map((version) => (
          <Version version={version} />
        ))}
      </div>
    ) : (
      <div
        className="inline-flex items-center place-self-center px-4 py-20
        leading-6 text-lg text-center text-slate-600
        transition ease-in-out duration-150"
      >
        <svg className="animate-spin h-5 w-5 mr-3" viewBox="0 0 24 24">
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
          ></circle>
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path>
        </svg>
        Loading...
      </div>
    );
  }
}

export default Versions;
