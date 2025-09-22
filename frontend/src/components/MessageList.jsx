import React, { useEffect, useRef } from 'react';

const MessageList = ({ messages }) => {
    const messagesEndRef = useRef(null);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    };

    useEffect(() => {
        scrollToBottom();
    }, [messages]);

    return (
        <ul id="messages">
            {messages.map((msg, index) => (
                <li key={index} className={msg.type}>
                    {msg.content}
                </li>
            ))}
            <div ref={messagesEndRef} />
        </ul>
    );
};

export default MessageList;
