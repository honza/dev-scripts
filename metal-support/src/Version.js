import React, { Component } from "react";

class Version extends Component {
  render() {
    return (
      <div className="p-8 bg-white rounded-lg shadow-lg font-mono">
        <h1 className="text-center font-bold text-4xl text-slate-600 mb-10">
          4.11
        </h1>
        <div className="text-sm grid grid-cols-2 gap-2">
          <div className="p-1 px-3 border-2 rounded-lg bg-green-400 hover:bg-green-300 border-green-600">
            Name
          </div>
          <div className="p-1 px-3 border-2 rounded-lg bg-red-400 hover:bg-red-300 border-red-600">
            Name
          </div>
        </div>
      </div>
    );
  }
}

export default Version;
