# Мой Петропавловск - Партисипаторное бюджетирование

## Overview
"Мой Петропавловск" is a full-featured participatory budgeting platform designed for the city of Petropavlovsk. Its primary purpose is to empower citizens to propose, discuss, and vote on urban development projects, fostering community engagement and transparent allocation of city resources. The platform aims to improve the urban environment through citizen-led initiatives, with key capabilities including user management, project submission with geographic tagging, AI-assisted moderation, intelligent voting, and interactive mapping. The platform seeks to enhance civic participation and streamline the budgeting process for local improvements.

## User Preferences
I prefer the AI to act as a coding assistant, helping me to develop and refine the application. Please provide clear, concise explanations and suggest improvements or new features based on best practices and the existing architecture. When making changes, prioritize using HTMX for dynamic updates and ensure the solution is performant and scalable. Avoid making major architectural changes without prior discussion. Focus on iterative development, providing solutions that I can review and integrate step-by-step.

## System Architecture
The platform is built with a Go backend using the Chi router, an HTMX + TailwindCSS frontend for dynamic and responsive UI, and PostgreSQL as the database. Key architectural decisions include:

-   **UI/UX**: Responsive design using TailwindCSS, HTMX for dynamic content updates without full page reloads, interactive Leaflet.js maps for project visualization and location selection, and clear empty states to guide users. Navigation is adaptive, featuring a horizontal menu for desktops and a smooth animated burger menu for mobile, implemented with Alpine.js.
-   **Technical Implementations**:
    -   **User Management**: Secure registration/login with email/password validation, HTTP-only cookie-based sessions, and protected routes.
    -   **Project Submission**: Form for project ideas including title, description (min 500 chars), category, district, budget, map coordinates via Leaflet, and image uploads (1-3 photos, JPG/PNG, max 5MB). Images are stored locally in `/uploads/{projectID}/`.
    -   **AI Moderation**: Google Gemini 1.5 Flash assists administrators by analyzing project ideas and comments. For projects, it identifies "pros" and "cons" (e.g., public benefit, budget realism vs. unrealistic budget, short description, toxicity) to inform admin decisions (approve, reject, edit). For comments, it validates for constructive feedback, rejecting short/toxic/non-substantive entries. The AI serves as an advisor, not a decision-maker.
    -   **Voting System**: Users can vote on projects, with mandatory comments (min 200 chars) validated by Gemini AI for constructiveness. One vote per project per user.
    -   **Interactive Map**: Displays all projects with color-coded markers based on status (Grey: Voting, Orange: In Progress, Green: Completed). HTMX loads project details into popup cards.
    -   **Project Lifecycle**: Projects transition through `moderation`, `voting`, `selected`, `in_progress`, `done`, or `rejected` statuses.
    -   **Gamification**: Comprehensive achievement and title system with automatic unlocking:
        - **Titles**: Automatically assigned based on user activity (Новичок → Активный житель → Идейный вдохновитель → Лидер мнений → Эксперт городского развития → Архитектор города)
        - **10 Achievements**: Automatically unlocked when conditions are met (registration, project submission, voting milestones, approved projects, wins, commenting)
        - **Statistics Dashboard**: Profile displays votes cast, ideas submitted, approved ideas, wins, and comments
        - **Database**: user_achievements table tracks unlocked achievements per user
        - **Auto-Check System**: Achievements validated after every user action (vote, submit, comment, admin status change)
    -   **File Storage**: Built-in system saves uploaded files to the `/uploads` directory.
-   **System Design Choices**:
    -   **Backend**: Go with Chi router provides a performant and lightweight server.
    -   **Database**: PostgreSQL for robust and scalable data storage, with tables for users, projects, votes, comments, and project status history.
    -   **Modular Structure**: Code is organized into `handlers`, `db`, `models`, `auth`, `ai`, `storage`, and `middleware` packages for maintainability.
    -   **Environment Configuration**: Utilizes environment variables for `DATABASE_URL`, `SESSION_SECRET`, and `GEMINI_API_KEY`.

## External Dependencies
-   **Database**: PostgreSQL
-   **AI Moderation**: Google Gemini 1.5 Flash (via `GEMINI_API_KEY`)
-   **Mapping**: Leaflet.js
-   **Frontend Interactivity**: HTMX, Alpine.js
-   **Styling**: TailwindCSS