import './App.scss';
import React from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import Controller from './Controller';
import Graph from './Graph';
import 'bootstrap';

export default class App extends React.Component {
    constructor(props) {
        super(props);
        this.mount = false;
    }

    render() {
        return (
            <>
                <header className={'navbar navbar-expand-lg sticky-top'}>
                    <div className={'container'}>
                        <a className={'navbar-brand d-flex flex-row'} href={'/'}>
                            <h3 className={'ms-3 mb-0 title'}>DAG-SSLE protocol controller</h3>
                        </a>
                        <button
                            className={'navbar-toggler'}
                            type={'button'}
                            data-bs-toggle={'collapse'}
                            data-bs-target={'#navbarNav'}
                            aria-controls={'navbarNav'}
                            aria-expanded={'false'}
                            aria-label={'Toggle navigation'}
                        >
                            <span className="navbar-toggler-icon"></span>
                        </button>
                        <div className="collapse navbar-collapse" id="navbarNav">
                            <ul className="navbar-nav flex-row flex-wrap bd-navbar-nav pt-2 py-md-0 w-100">
                                <a className="ms-auto pt-2 px-2 py-md-0" href="/control">
                                    <li className="nav-item px-2">Control</li>
                                </a>
                                <a className="pt-2 px-2 py-md-0" href="/graph">
                                <li className="nav-item px-2">Graph</li>
                                </a>
                            </ul>
                        </div>
                    </div>
                </header>
                <div className="container center text-center my-5">
                    <BrowserRouter>
                        <Routes>
                            <Route index exact path={'/'} element={<Controller />}></Route>
                            <Route exact path={'/control'} element={<Controller />}></Route>
                            <Route exact path={'/graph'} element={<Graph />}></Route>
                        </Routes>
                    </BrowserRouter>
                </div>
                <footer className="mb-3 text-center">
                    Tomas Hladky, part of the Master thesis (Design an Experimental PoS DAG-based Blockchain Consensual Protocol) at BUT at 2025.{' '}
                </footer>
            </>
        );
    }
}
