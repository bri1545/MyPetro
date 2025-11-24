package handlers

import (
        "encoding/json"
        "fmt"
        "html/template"
        "net/http"
        "petropavlovsk-budget/internal/ai"
        "petropavlovsk-budget/internal/auth"
        "petropavlovsk-budget/internal/db"
        "petropavlovsk-budget/internal/models"
        "petropavlovsk-budget/internal/storage"
        "strconv"
        "strings"

        "github.com/go-chi/chi/v5"
        "github.com/gorilla/sessions"
)

type Handler struct {
        DB        *db.Database
        Store     *sessions.CookieStore
        Templates *template.Template
}

func New(database *db.Database, store *sessions.CookieStore) *Handler {
        tmpl := template.Must(template.ParseGlob("templates/*.html"))
        return &Handler{
                DB:        database,
                Store:     store,
                Templates: tmpl,
        }
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]
        userRole := session.Values["role"]

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
                "IsAdmin":  userRole == "admin",
        }

        h.Templates.ExecuteTemplate(w, "index.html", data)
}

func (h *Handler) RegisterPage(w http.ResponseWriter, r *http.Request) {
        h.Templates.ExecuteTemplate(w, "register.html", nil)
}

func (h *Handler) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
        email := r.FormValue("email")
        password := r.FormValue("password")
        confirmPassword := r.FormValue("confirm_password")

        if password != confirmPassword {
                w.Header().Set("HX-Retarget", "#error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Пароли не совпадают</div>`))
                return
        }

        if err := auth.ValidatePassword(password); err != nil {
                w.Header().Set("HX-Retarget", "#error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(fmt.Sprintf(`<div class="text-red-600 text-sm">%s</div>`, err.Error())))
                return
        }

        hash, err := auth.HashPassword(password)
        if err != nil {
                w.Header().Set("HX-Retarget", "#error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Ошибка сервера</div>`))
                return
        }

        user, err := h.DB.CreateUser(email, hash)
        if err != nil {
                w.Header().Set("HX-Retarget", "#error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Email уже зарегистрирован</div>`))
                return
        }

        session, _ := h.Store.Get(r, "session")
        session.Values["user_id"] = user.ID
        session.Values["email"] = user.Email
        session.Values["role"] = user.Role
        session.Save(r, w)

        w.Header().Set("HX-Redirect", "/")
        w.WriteHeader(http.StatusOK)
}

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
        h.Templates.ExecuteTemplate(w, "login.html", nil)
}

func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
        email := r.FormValue("email")
        password := r.FormValue("password")

        user, err := h.DB.GetUserByEmail(email)
        if err != nil {
                w.Header().Set("HX-Retarget", "#error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Неверный email или пароль</div>`))
                return
        }

        if err := auth.CheckPassword(password, user.PasswordHash); err != nil {
                w.Header().Set("HX-Retarget", "#error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Неверный email или пароль</div>`))
                return
        }

        session, _ := h.Store.Get(r, "session")
        session.Values["user_id"] = user.ID
        session.Values["email"] = user.Email
        session.Values["role"] = user.Role
        session.Save(r, w)

        if user.Role == "admin" {
                w.Header().Set("HX-Redirect", "/admin")
        } else {
                w.Header().Set("HX-Redirect", "/")
        }
        w.WriteHeader(http.StatusOK)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        session.Options.MaxAge = -1
        session.Save(r, w)
        http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) SubmitPage(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]
        userRole := session.Values["role"]

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
                "IsAdmin":  userRole == "admin",
        }

        h.Templates.ExecuteTemplate(w, "submit.html", data)
}

func (h *Handler) SubmitProject(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]
        if userID == nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
        }

        r.ParseMultipartForm(20 << 20)

        title := r.FormValue("title")
        description := r.FormValue("description")
        category := r.FormValue("category")
        district := r.FormValue("district")
        budgetStr := r.FormValue("budget")
        latStr := r.FormValue("lat")
        lngStr := r.FormValue("lng")

        budget, _ := strconv.Atoi(budgetStr)
        lat, _ := strconv.ParseFloat(latStr, 64)
        lng, _ := strconv.ParseFloat(lngStr, 64)

        submission := models.ProjectSubmission{
                Title:       title,
                Description: description,
                Category:    category,
                District:    district,
                Budget:      budget,
                Lat:         lat,
                Lng:         lng,
        }

        result := ai.ValidateIdeaWithGemini(submission)
        if !result.Approved {
                w.Header().Set("HX-Retarget", "#error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(fmt.Sprintf(`<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded"><strong>Проект отклонён ИИ:</strong> %s</div>`, result.Reason)))
                return
        }

        project := &models.Project{
                Title:       title,
                Description: description,
                Category:    category,
                District:    district,
                Budget:      budget,
                Lat:         lat,
                Lng:         lng,
                Status:      "moderation",
                UserID:      userID.(int),
                Images:      []string{},
        }

        err := h.DB.CreateProject(project)
        if err != nil {
                w.Header().Set("HX-Retarget", "#error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Ошибка создания проекта</div>`))
                return
        }

        files := r.MultipartForm.File["photos"]
        if len(files) > 0 {
                imagePaths, err := storage.SaveProjectImages(project.ID, files)
                if err != nil {
                        w.Header().Set("HX-Retarget", "#error")
                        w.Header().Set("HX-Reswap", "innerHTML")
                        w.Write([]byte(fmt.Sprintf(`<div class="text-red-600 text-sm">%s</div>`, err.Error())))
                        return
                }
                project.Images = imagePaths
        }

        w.Header().Set("HX-Redirect", "/projects")
        w.WriteHeader(http.StatusOK)
}

func (h *Handler) ProjectsPage(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]
        userRole := session.Values["role"]

        projects, err := h.DB.GetAllProjects()
        if err != nil {
                projects = []models.Project{}
        }

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
                "IsAdmin":  userRole == "admin",
                "Projects": projects,
        }

        h.Templates.ExecuteTemplate(w, "projects.html", data)
}

func (h *Handler) ProjectDetail(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]
        userRole := session.Values["role"]

        projectIDStr := chi.URLParam(r, "id")
        projectID, _ := strconv.Atoi(projectIDStr)

        project, err := h.DB.GetProjectByID(projectID)
        if err != nil {
                http.Error(w, "Проект не найден", http.StatusNotFound)
                return
        }

        votes, _ := h.DB.GetProjectVotes(projectID)
        comments, _ := h.DB.GetProjectComments(projectID)
        history, _ := h.DB.GetProjectStatusHistory(projectID)

        hasVoted := false
        if userID != nil {
                hasVoted, _ = h.DB.HasUserVoted(projectID, userID.(int))
        }

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
                "IsAdmin":  userRole == "admin",
                "Project":  project,
                "Votes":    votes,
                "Comments": comments,
                "History":  history,
                "HasVoted": hasVoted,
        }

        h.Templates.ExecuteTemplate(w, "project_detail.html", data)
}

func (h *Handler) VoteSubmit(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]
        if userID == nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
        }

        projectIDStr := r.FormValue("project_id")
        comment := r.FormValue("comment")

        projectID, _ := strconv.Atoi(projectIDStr)

        hasVoted, _ := h.DB.HasUserVoted(projectID, userID.(int))
        if hasVoted {
                w.Header().Set("HX-Retarget", "#vote-error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Вы уже проголосовали за этот проект</div>`))
                return
        }

        valid, reason := ai.ValidateVoteCommentWithGemini(comment)
        if !valid {
                w.Header().Set("HX-Retarget", "#vote-error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(fmt.Sprintf(`<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded"><strong>Комментарий отклонён ИИ:</strong> %s</div>`, reason)))
                return
        }

        err := h.DB.CreateVote(projectID, userID.(int), comment)
        if err != nil {
                w.Header().Set("HX-Retarget", "#vote-error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Ошибка при сохранении голоса</div>`))
                return
        }

        w.Header().Set("HX-Redirect", fmt.Sprintf("/projects/%d", projectID))
        w.WriteHeader(http.StatusOK)
}

func (h *Handler) MapPage(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]
        userRole := session.Values["role"]

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
                "IsAdmin":  userRole == "admin",
        }

        h.Templates.ExecuteTemplate(w, "map.html", data)
}

func (h *Handler) MapData(w http.ResponseWriter, r *http.Request) {
        projects, err := h.DB.GetAllProjects()
        if err != nil {
                w.WriteHeader(http.StatusInternalServerError)
                json.NewEncoder(w).Encode([]models.Project{})
                return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(projects)
}

func (h *Handler) ProjectPopup(w http.ResponseWriter, r *http.Request) {
        projectIDStr := chi.URLParam(r, "id")
        projectID, _ := strconv.Atoi(projectIDStr)

        project, err := h.DB.GetProjectByID(projectID)
        if err != nil {
                http.Error(w, "Проект не найден", http.StatusNotFound)
                return
        }

        statusText := map[string]string{
                "voting":      "Голосование",
                "in_progress": "В работе",
                "done":        "Завершён",
        }

        statusColor := map[string]string{
                "voting":      "bg-gray-200 text-gray-800",
                "in_progress": "bg-orange-200 text-orange-800",
                "done":        "bg-green-200 text-green-800",
        }

        html := fmt.Sprintf(`
                <div class="p-4 max-w-sm">
                        <h3 class="font-bold text-lg mb-2">%s</h3>
                        <p class="text-sm text-gray-600 mb-2"><span class="px-2 py-1 rounded %s">%s</span></p>
                        <p class="text-sm mb-2">%s</p>
                        <p class="text-sm mb-2"><strong>Бюджет:</strong> %s ₸</p>
                        <p class="text-sm mb-2"><strong>Голосов:</strong> %d</p>
                        <a href="/projects/%d" class="text-blue-600 hover:underline text-sm">Подробнее →</a>
                </div>
        `, project.Title, statusColor[project.Status], statusText[project.Status], 
           truncateString(project.Description, 100), formatNumber(project.Budget), 
           project.VoteCount, project.ID)

        w.Write([]byte(html))
}

func truncateString(s string, maxLen int) string {
        if len(s) <= maxLen {
                return s
        }
        return s[:maxLen] + "..."
}

func formatNumber(n int) string {
        s := strconv.Itoa(n)
        var result []string
        for i, c := range reverse(s) {
                if i > 0 && i%3 == 0 {
                        result = append(result, " ")
                }
                result = append(result, string(c))
        }
        return reverse(strings.Join(result, ""))
}

func reverse(s string) string {
        runes := []rune(s)
        for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
                runes[i], runes[j] = runes[j], runes[i]
        }
        return string(runes)
}

func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]
        userRole := session.Values["role"]

        moderationProjects, _ := h.DB.GetProjectsByStatus("moderation")
        votingProjects, _ := h.DB.GetProjectsByStatus("voting")
        selectedProjects, _ := h.DB.GetProjectsByStatus("selected")
        inProgressProjects, _ := h.DB.GetProjectsByStatus("in_progress")
        doneProjects, _ := h.DB.GetProjectsByStatus("done")

        data := map[string]interface{}{
                "LoggedIn":           userID != nil,
                "IsAdmin":            userRole == "admin",
                "ModerationProjects": moderationProjects,
                "VotingProjects":     votingProjects,
                "SelectedProjects":   selectedProjects,
                "InProgressProjects": inProgressProjects,
                "DoneProjects":       doneProjects,
        }

        h.Templates.ExecuteTemplate(w, "admin.html", data)
}

func (h *Handler) AdminUpdateProjectStatus(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        adminID := session.Values["user_id"].(int)

        projectIDStr := r.FormValue("project_id")
        projectID, _ := strconv.Atoi(projectIDStr)
        newStatus := r.FormValue("status")
        comment := r.FormValue("comment")

        err := h.DB.UpdateProjectStatus(projectID, newStatus, adminID, comment)
        if err != nil {
                w.Header().Set("HX-Retarget", "#error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Ошибка обновления статуса</div>`))
                return
        }

        if newStatus == "voting" {
                voteStart := r.FormValue("vote_start")
                voteEnd := r.FormValue("vote_end")
                if voteStart != "" && voteEnd != "" {
                        h.DB.SetVotingPeriod(projectID, voteStart, voteEnd)
                }
        }

        w.Header().Set("HX-Redirect", "/admin")
        w.WriteHeader(http.StatusOK)
}

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]
        if userID == nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
        }

        projectIDStr := r.FormValue("project_id")
        projectID, _ := strconv.Atoi(projectIDStr)
        content := r.FormValue("content")

        if len(content) < 50 {
                w.Header().Set("HX-Retarget", "#comment-error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Комментарий должен быть минимум 50 символов</div>`))
                return
        }

        err := h.DB.CreateComment(projectID, userID.(int), content)
        if err != nil {
                w.Header().Set("HX-Retarget", "#comment-error")
                w.Header().Set("HX-Reswap", "innerHTML")
                w.Write([]byte(`<div class="text-red-600 text-sm">Ошибка добавления комментария</div>`))
                return
        }

        w.Header().Set("HX-Refresh", "true")
        w.WriteHeader(http.StatusOK)
}
