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

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
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
        session.Save(r, w)

        w.Header().Set("HX-Redirect", "/")
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

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
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
                Status:      "voting",
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

        projects, err := h.DB.GetAllProjects()
        if err != nil {
                projects = []models.Project{}
        }

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
                "Projects": projects,
        }

        h.Templates.ExecuteTemplate(w, "projects.html", data)
}

func (h *Handler) ProjectDetail(w http.ResponseWriter, r *http.Request) {
        session, _ := h.Store.Get(r, "session")
        userID := session.Values["user_id"]

        projectIDStr := chi.URLParam(r, "id")
        projectID, _ := strconv.Atoi(projectIDStr)

        project, err := h.DB.GetProjectByID(projectID)
        if err != nil {
                http.Error(w, "Проект не найден", http.StatusNotFound)
                return
        }

        votes, _ := h.DB.GetProjectVotes(projectID)

        hasVoted := false
        if userID != nil {
                hasVoted, _ = h.DB.HasUserVoted(projectID, userID.(int))
        }

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
                "Project":  project,
                "Votes":    votes,
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

        data := map[string]interface{}{
                "LoggedIn": userID != nil,
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
