# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gwent Classic is a browser-based implementation of the Gwent card game from The Witcher 3. It's a pure frontend JavaScript application with no build process or dependencies.

## Development Commands

Since this is a static frontend application, there are no build/test/lint commands. To run the application:
1. Open `index.html` in a web browser
2. Use a local web server if needed: `python -m http.server 8080` 

## Architecture

### Core Components

**gwent.js** - Main game engine containing:
- `Game` class - Core game state management and turn logic
- `AI` class - Opponent AI implementation with decision-making algorithms
- `Board` class - Game board state and card placement logic
- `Card` class - Individual card representation and behavior
- Event handlers for user interactions

**cards.js** - Complete card database:
- JSON structure with all cards from base game + DLCs
- Each card has: name, strength, abilities, row placement, faction

**abilities.js** - Card ability implementations:
- Special abilities like Spy, Medic, Scorch, Tight Bond
- Weather effects and their interactions
- Leader abilities for each faction

**factions.js** - Faction-specific logic:
- Leader abilities
- Faction passive abilities
- Faction-specific card interactions

### Game Flow

1. Initialization loads cards database and sets up board
2. Players take turns playing cards or passing
3. AI evaluates board state and makes optimal moves
4. Round ends when both players pass
5. Best of 3 rounds determines winner

### Key Implementation Details

- AI uses minimax-like evaluation with heuristics for card value, board control, and future potential
- Card abilities are implemented as functions that modify game state
- Save/load functionality uses localStorage for deck persistence
- No server-side code - everything runs in the browser

## Containerization

The application is containerized with:
- Nginx Alpine image serving static files
- Kubernetes manifests for deployment (deployment.yaml, service.yaml)
- Published image: `gwynbliedd/gwent-game:v0.1`

## When Making Changes

1. Test all card interactions manually in the browser
2. Ensure AI still makes reasonable decisions after logic changes
3. Verify deck import/export functionality remains intact
4. Check that all card images load correctly
5. Test in multiple browsers for compatibility

## Project Goals - 24 July

- Implement a server-side (Written in Golang) instead of having everything loaded to browser
- Having a postgreSQL as a database for the project
- Game Flow and Core remain the same
- Having a multiplayer instead of facing only AI opponent
- Having a Login/Register mechanism
- Do not be bothered with docker nor kubernetes, those will be edited manually in future


## Future expectations - 24 July

- When someone enter the browser, he can login or register. it will be saved in database for future vists
- When he registerd with an email, name and password. User will have an ID
- After login, the player must have an option to either challenge AI bot or Challenge another player by entering his name#ID


## Reached progress by end of 24 July

- DB is set
- Backend server is set
- CORS issue resolved
- frontend updated to use backend


## current issues noticed: 25 July

- when X player sends a challenge to Y, it doesnt appear on Y user browser until he refresh page, it should be dynamic and not require refreshing
- when Y player accept the challeng (after he refresh his page to see the challenge request), he is directed to the game at the moment. Meanwhile X player needs to refresh and see the active games window and press on it!!
- However, when Y Player accept challenge HE IS ACTUALLY BEEN DIRECTED TO PLAY WITH A COMPUTER!! not a real player!!
- Also, when X player saw that his challenge accepted. He saw an active game -> clicks on it -> then he is directed to a game also vs a computer!!!