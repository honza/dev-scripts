import React, { Component } from "react";
import axios from "axios";

import Versions from "./Versions";

export default class App extends Component {
  state = {
    versions: [],
    loaded: false,
    lastUpdated: null,
  };

  componentDidMount() {
    this.fetchData();
    this.timer = setInterval(() => this.fetchData(), 60 * 1000);
  }

  fetchData() {
    this.setState({ loaded: false });

    axios.get(`/data.json`).then((res) => {
      const versions = res.data.versions;
      const lastUpdated = res.data.last_updated;
      this.setState({ versions, lastUpdated, loaded: true });
    });
  }

  render() {
    return (
      <div className="bg-slate-100 font-mono">
        <div className="bg-white mb-1 h-10">
          <h1 className="text-lg p-3">Metal Wall</h1>
        </div>

        <Versions loaded={this.state.loaded} versions={this.state.versions} />

        <div className="text-xs text-center text-slate-400 mt-10">
          <p>Last updated {this.state.lastUpdated}</p>
        </div>
      </div>
    );
  }
}
