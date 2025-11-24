package achievements

import "petropavlovsk-budget/internal/models"

var AllAchievements = map[string]models.Achievement{
        "newcomer": {
                ID:          "newcomer",
                Title:       "Новичок",
                Description: "Добро пожаловать на платформу!",
                Icon:        "🎯",
                Requirement: 0,
        },
        "first_project": {
                ID:          "first_project",
                Title:       "Первый шаг",
                Description: "Подана первая идея проекта",
                Icon:        "💡",
                Requirement: 1,
        },
        "voter": {
                ID:          "voter",
                Title:       "Голос народа",
                Description: "Проголосовал за 5 проектов",
                Icon:        "🗳️",
                Requirement: 5,
        },
        "active_citizen": {
                ID:          "active_citizen",
                Title:       "Активный житель",
                Description: "Проголосовал за 10 проектов",
                Icon:        "⭐",
                Requirement: 10,
        },
        "idea_inspirer": {
                ID:          "idea_inspirer",
                Title:       "Идейный вдохновитель",
                Description: "3 идеи одобрены модератором",
                Icon:        "🎯",
                Requirement: 3,
        },
        "city_architect": {
                ID:          "city_architect",
                Title:       "Архитектор города",
                Description: "Одна из ваших идей победила в голосовании!",
                Icon:        "🏗️",
                Requirement: 1,
        },
        "opinion_leader": {
                ID:          "opinion_leader",
                Title:       "Лидер мнений",
                Description: "Проголосовал за 25 проектов",
                Icon:        "👑",
                Requirement: 25,
        },
        "expert": {
                ID:          "expert",
                Title:       "Эксперт",
                Description: "5 идей одобрены модератором",
                Icon:        "🌟",
                Requirement: 5,
        },
        "commentator": {
                ID:          "commentator",
                Title:       "Комментатор",
                Description: "Оставлено 10 комментариев",
                Icon:        "💬",
                Requirement: 10,
        },
        "discussant": {
                ID:          "discussant",
                Title:       "Обсуждатель",
                Description: "Оставлено 25 комментариев",
                Icon:        "🗣️",
                Requirement: 25,
        },
}

func GetAchievement(id string) (models.Achievement, bool) {
        achievement, exists := AllAchievements[id]
        return achievement, exists
}

func GetAllAchievementsList() []models.Achievement {
        achievements := make([]models.Achievement, 0, len(AllAchievements))
        for _, achievement := range AllAchievements {
                achievements = append(achievements, achievement)
        }
        return achievements
}
