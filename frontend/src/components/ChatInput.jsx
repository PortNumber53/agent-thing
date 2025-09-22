import React, { useState } from 'react';

const ChatInput = ({ onSend }) => {
    const [inputValue, setInputValue] = useState('');

    const handleSubmit = (event) => {
        event.preventDefault();
        if (inputValue) {
            onSend(inputValue);
            setInputValue('');
        }
    };

    return (
        <form id="form" onSubmit={handleSubmit}>
            <input
                id="input"
                autoComplete="off"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                placeholder="Enter a task for the agent..."
            />
            <button type="submit">Send</button>
        </form>
    );
};

export default ChatInput;
