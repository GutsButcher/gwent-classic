const API_URL = 'http://localhost:8080/api';

class GwentAPI {
    constructor() {
        this.token = localStorage.getItem('gwent_token');
        this.user = JSON.parse(localStorage.getItem('gwent_user') || 'null');
        this.ws = null;
        this.gameId = null;
    }

    getAuthHeaders() {
        return {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${this.token}`
        };
    }

    async login(email, password) {
        const response = await fetch(`${API_URL}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });
        
        if (!response.ok) throw new Error('Login failed');
        
        const data = await response.json();
        this.token = data.token;
        this.user = data.user;
        localStorage.setItem('gwent_token', data.token);
        localStorage.setItem('gwent_user', JSON.stringify(data.user));
        return data;
    }

    async register(email, name, password) {
        const response = await fetch(`${API_URL}/auth/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, name, password })
        });
        
        if (!response.ok) throw new Error('Registration failed');
        
        const data = await response.json();
        this.token = data.token;
        this.user = data.user;
        localStorage.setItem('gwent_token', data.token);
        localStorage.setItem('gwent_user', JSON.stringify(data.user));
        return data;
    }

    logout() {
        this.token = null;
        this.user = null;
        localStorage.removeItem('gwent_token');
        localStorage.removeItem('gwent_user');
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        window.location.href = 'auth.html';
    }

    isAuthenticated() {
        return this.token && this.user;
    }

    async createAIGame() {
        const response = await fetch(`${API_URL}/protected/games/ai`, {
            method: 'POST',
            headers: this.getAuthHeaders()
        });
        
        if (!response.ok) throw new Error('Failed to create AI game');
        
        const data = await response.json();
        this.gameId = data.game_id;
        return data;
    }

    async createChallenge(challengedNameID) {
        const response = await fetch(`${API_URL}/protected/challenges`, {
            method: 'POST',
            headers: this.getAuthHeaders(),
            body: JSON.stringify({ challenged_name_id: challengedNameID })
        });
        
        if (!response.ok) throw new Error('Failed to create challenge');
        return await response.json();
    }

    async getChallenges() {
        const response = await fetch(`${API_URL}/protected/challenges`, {
            headers: this.getAuthHeaders()
        });
        
        if (!response.ok) throw new Error('Failed to get challenges');
        return await response.json();
    }

    async respondToChallenge(challengeId, accept) {
        const response = await fetch(`${API_URL}/protected/challenges/${challengeId}/respond`, {
            method: 'POST',
            headers: this.getAuthHeaders(),
            body: JSON.stringify({ accept })
        });
        
        if (!response.ok) throw new Error('Failed to respond to challenge');
        return await response.json();
    }

    async getActiveGames() {
        const response = await fetch(`${API_URL}/protected/games`, {
            headers: this.getAuthHeaders()
        });
        
        if (!response.ok) throw new Error('Failed to get active games');
        return await response.json();
    }

    async searchUser(nameID) {
        const response = await fetch(`${API_URL}/users/search?nameID=${encodeURIComponent(nameID)}`, {
            headers: this.getAuthHeaders()
        });
        
        if (!response.ok) throw new Error('User not found');
        return await response.json();
    }

    connectToGame(gameId, onMessage) {
        if (this.ws) {
            this.ws.close();
        }
        
        this.gameId = gameId;
        const wsUrl = `ws://localhost:8080/ws/game/${gameId}?token=${this.token}`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('Connected to game', gameId);
            this.ws.send(JSON.stringify({
                type: 'game_state_request',
                game_id: gameId
            }));
        };
        
        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (onMessage) {
                onMessage(data);
            }
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
        
        this.ws.onclose = () => {
            console.log('Disconnected from game');
        };
        
        return this.ws;
    }

    sendGameMove(move) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('WebSocket not connected');
            return;
        }
        
        this.ws.send(JSON.stringify({
            type: 'move',
            game_id: this.gameId,
            user_id: this.user.id,
            payload: move
        }));
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
}

// Export as global variable for use in existing code
window.gwentAPI = new GwentAPI();