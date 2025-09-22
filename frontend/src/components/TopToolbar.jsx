import React from 'react';
import './TopToolbar.css';

const TopToolbar = ({
    onGenerate,
    onDownloadPublic,
    onDownloadPrivate,
    onCopyPublic,
    onCopyPrivate,
    onContainerStart,
    onContainerStop,
    onContainerRebuild,
    onContainerStatus,
}) => {
    return (
        <div id="top-toolbar">
            <div className="dropdown">
                <button className="dropdown-button">SSH Keys</button>
                <div className="dropdown-content">
                    <button onClick={onGenerate}>Generate ed25519 Key</button>
                    <button onClick={onDownloadPublic}>Download Public Key</button>
                    <button onClick={onDownloadPrivate}>Download Private Key</button>
                    <button onClick={onCopyPublic}>Copy Public Key</button>
                    <button onClick={onCopyPrivate}>Copy Private Key</button>
                </div>
            </div>
            <div className="dropdown">
                <button className="dropdown-button">Container</button>
                <div className="dropdown-content">
                    <button onClick={onContainerStart}>Start</button>
                    <button onClick={onContainerStop}>Stop</button>
                    <button onClick={onContainerRebuild}>Rebuild</button>
                    <button onClick={onContainerStatus}>Status</button>
                </div>
            </div>
        </div>
    );
};

export default TopToolbar;
