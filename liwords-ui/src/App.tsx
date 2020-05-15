import React, { useLayoutEffect, useState } from 'react';
import './App.css';
import { Table } from './gameroom/table';
import 'antd/dist/antd.css';

function useWindowSize() {
  const [size, setSize] = useState([0, 0]);
  useLayoutEffect(() => {
    function updateSize() {
      setSize([window.innerWidth, window.innerHeight]);
    }
    window.addEventListener('resize', updateSize);
    updateSize();
    return () => window.removeEventListener('resize', updateSize);
  }, []);
  return size;
}

const App = () => {
  const [width, _] = useWindowSize();
  return (
    <div className="App">
      <Table windowWidth={width} />
    </div>
  );
};

export default App;
