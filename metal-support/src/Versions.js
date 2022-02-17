import React, { Component } from "react";
import Version from "./Version";

class Versions extends Component {
  render() {
    return (
      <div className="flex flex-row items-stretch gap-4 w-full p-3">
        <Version />
      </div>
    );
  }
}

export default Versions;
