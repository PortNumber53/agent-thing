import React, { useState, useRef, useEffect } from 'react';

const ChatInput = ({ onSend }) => {
    const [inputValue, setInputValue] = useState('');
    const textareaRef = useRef(null);

    const handleSubmit = (event) => {
        event.preventDefault();
        if (inputValue.trim()) {
            onSend(inputValue.trim());
            setInputValue('');
        }
    };

    const handleKeyDown = (event) => {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            handleSubmit(event);
        }
    };

    // Auto-resize textarea height
    useEffect(() => {
        if (textareaRef.current) {
            textareaRef.current.style.height = 'auto'; // Reset height
            textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`;
        }
    }, [inputValue]);

    return (
        <form id="form" onSubmit={handleSubmit}>
            <textarea
                ref={textareaRef}
                id="input"
                autoComplete="off"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Enter a task for the agent... (Shift+Enter for new line)"
                rows="1"
                style={{ 
                    resize: 'none',
                    maxHeight: '150px', // Set a max height
                 }}
            />
            <button type="submit">Send</button>
        </form>
    );
};

export default ChatInput;
