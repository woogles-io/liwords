import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import App from './App';
import * as serviceWorker from './serviceWorker';

const username = prompt('what is ur username');
if (!username) {
  console.log('erorr no username');
} else {
  ReactDOM.render(
    <React.StrictMode>
      <App username={username} />
    </React.StrictMode>,
    document.getElementById('root')
  );
}

// If you want your app to work offline and load faster, you can change
// unregister() to register() below. Note this comes with some pitfalls.
// Learn more about service workers: https://bit.ly/CRA-PWA
serviceWorker.unregister();
