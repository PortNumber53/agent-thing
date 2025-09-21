document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('form');
    const input = document.getElementById('input');
    const messages = document.getElementById('messages');

    const ws = new WebSocket(`ws://${window.location.host}/ws`);

    const addMessage = (content, type) => {
        const item = document.createElement('li');
        item.textContent = content;
        item.classList.add(type);
        messages.appendChild(item);
        messages.scrollTop = messages.scrollHeight;
    };

    ws.onopen = () => {
        addMessage('Connected to the agent.', 'status-message');
    };

    ws.onmessage = (event) => {
        addMessage(event.data, 'agent-message');
    };

    ws.onclose = () => {
        addMessage('Connection closed.', 'status-message');
    };

    ws.onerror = (error) => {
        addMessage('An error occurred.', 'status-message');
        console.error('WebSocket Error:', error);
    };

    form.addEventListener('submit', (event) => {
        event.preventDefault();
        if (input.value) {
            addMessage(input.value, 'user-message');
            ws.send(input.value);
            input.value = '';
        }
    });
});
