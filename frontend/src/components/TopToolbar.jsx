import React from 'react';

const TopToolbar = ({ onGenerate, onDownloadPublic, onDownloadPrivate, onCopyPublic, onCopyPrivate }) => {
    return (
        <div id="top-toolbar">
            <button onClick={onGenerate}>Generate ed25519 SSH Key</button>
            <button onClick={onDownloadPublic}>Download Public Key</button>
            <button onClick={onDownloadPrivate}>Download Private Key</button>
            <button onClick={onCopyPublic}>Copy Public Key</button>
            <button onClick={onCopyPrivate}>Copy Private Key</button>
        </div>
    );
};

export default TopToolbar;
