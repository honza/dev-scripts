import Versions from "./Versions";

function App() {
  return (
    <div className="bg-slate-100 font-mono">
      <div className="bg-white mb-1 h-10">
        <h1 className="text-lg p-3">Metal Wall</h1>
      </div>

      <Versions />

      <div className="text-xs text-center text-slate-400 mt-10">
        <p>Last updated 2022-02-16 13:01:86</p>
      </div>
    </div>
  );
}

export default App;
