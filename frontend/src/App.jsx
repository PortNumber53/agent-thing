import React, { useState, useEffect, useRef } from 'react';
import TopToolbar from './components/TopToolbar';
import MessageList from './components/MessageList';
import ChatInput from './components/ChatInput';
import StatusBar from './components/StatusBar';
import './App.css';

function App() {
    const [messages, setMessages] = useState([]);
    const [status, setStatus] = useState('Connecting...');
    const ws = useRef(null);

    const addMessage = (content, type) => {
        setMessages(prev => [...prev, { content, type }]);
    };

    const connect = () => {
        const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
        const backendHost = import.meta.env.VITE_BACKEND_HOST || window.location.host;
        ws.current = new WebSocket(`${protocol}://${backendHost}/ws`);

        ws.current.onopen = () => {
            setStatus('Connected to the agent.');
        };

        ws.current.onmessage = (event) => {
            const data = event.data;
            if (data.startsWith('--- FILE_CONTENT ---')) {
                const parts = data.split('--- FILE_CONTENT ---');
                const fileInfo = JSON.parse(parts[1]);
                handleFileContent(fileInfo.name, fileInfo.content, fileInfo.action);
            } else {
                addMessage(data, 'agent-message');
            }
        };

        ws.current.onclose = () => {
            setStatus('Disconnected. Attempting to reconnect in 2 seconds...');
            setTimeout(connect, 2000);
        };

        ws.current.onerror = (error) => {
            setStatus('An error occurred.');
            console.error('WebSocket Error:', error);
            ws.current.close();
        };
    };

    useEffect(() => {
        connect();
        return () => {
            if (ws.current) {
                ws.current.close();
            }
        };
    }, []);

    const handleFileContent = (fileName, content, action) => {
        if (action === 'download') {
            const blob = new Blob([content], { type: 'text/plain' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = fileName;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
            setStatus(`Downloaded ${fileName}`);
        } else if (action === 'copy') {
            navigator.clipboard.writeText(content).then(() => {
                setStatus(`Copied ${fileName} to clipboard.`);
            }, (err) => {
                setStatus(`Failed to copy: ${err}`);
            });
        }
    };

    const sendAgentCommand = (command) => {
        if (ws.current && ws.current.readyState === WebSocket.OPEN) {
            addMessage(command, 'user-message');
            const message = {
                type: 'conversation',
                payload: command
            };
            ws.current.send(JSON.stringify(message));
        }
    };

    const sendToolCommand = (tool, args) => {
        if (ws.current && ws.current.readyState === WebSocket.OPEN) {
            const command = `${tool} ${args.join(' ')}`;
            addMessage(`Executing: ${command}`, 'user-message');
            const message = {
                type: 'tool_exec',
                payload: { tool, args }
            };
            ws.current.send(JSON.stringify(message));
        }
    };

    return (
        <div className="App">
            <TopToolbar
                onGenerate={() => sendToolCommand('ssh_key_gen', [])}
                onDownloadPublic={() => sendToolCommand('file_read', ['/home/developer/.ssh/id_ed25519.pub', 'for', 'download'])}
                onDownloadPrivate={() => sendToolCommand('file_read', ['/home/developer/.ssh/id_ed25519', 'for', 'download'])}
                onCopyPublic={() => sendToolCommand('file_read', ['/home/developer/.ssh/id_ed25519.pub', 'for', 'copy'])}
                onCopyPrivate={() => sendToolCommand('file_read', ['/home/developer/.ssh/id_ed25519', 'for', 'copy'])}
                onContainerStart={() => sendToolCommand('docker_start', [])}
                onContainerStop={() => sendToolCommand('docker_stop', [])}
                onContainerRebuild={() => sendToolCommand('docker_rebuild', [])}
                onContainerStatus={() => sendToolCommand('docker_status', [])}
            />
            <MessageList messages={messages} />
            <ChatInput onSend={sendAgentCommand} />
            <StatusBar status={status} />
        </div>
    );
}

export default App;
