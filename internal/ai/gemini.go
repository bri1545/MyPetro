package ai

import (
        "bytes"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "os"
        "petropavlovsk-budget/internal/models"
        "strings"
)

type GeminiRequest struct {
        Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
        Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
        Text string `json:"text"`
}

type GeminiResponse struct {
        Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
        Content GeminiContent `json:"content"`
}

type AIAnalysis struct {
        Pros []string
        Cons []string
}

func AnalyzeIdeaWithGemini(p models.ProjectSubmission) AIAnalysis {
        apiKey := os.Getenv("GEMINI_API_KEY")
        if apiKey == "" {
                return AIAnalysis{
                        Pros: []string{},
                        Cons: []string{"Ошибка конфигурации системы модерации"},
                }
        }

        prompt := fmt.Sprintf(`Ты эксперт по партисипаторному бюджетированию для города Петропавловск, Казахстан.

Проанализируй эту идею проекта и составь список плюсов и минусов для помощи администраторам в принятии решения.

НАЗВАНИЕ: %s
ОПИСАНИЕ: %s
КАТЕГОРИЯ: %s
РАЙОН: %s
БЮДЖЕТ: %d ₸
КООРДИНАТЫ: lat=%f, lng=%f

ПЛЮСЫ - что хорошо в проекте:
- Есть ли общественная польза?
- Улучшает ли городскую среду?
- Соответствует ли категории партисипаторного бюджетирования?
- Четко ли описана проблема и решение?
- Реалистичен ли бюджет?

МИНУСЫ - что может быть проблемой:
- Проблемы с бюджетом (слишком мало или много, не в пределах 300k-2M ₸)
- Проблемы с описанием (короткое, неясное)
- Отсутствие координат или общественной пользы
- Личное использование вместо общественного блага
- Токсичность или несоответствие категориям
- Любые другие проблемы

Ответь СТРОГО в формате JSON:
{
  "pros": ["плюс 1", "плюс 2", ...],
  "cons": ["минус 1", "минус 2", ...]
}

Каждый пункт должен быть кратким (одно предложение). Если плюсов или минусов нет - верни пустой массив.`, p.Title, p.Description, p.Category, p.District, p.Budget, p.Lat, p.Lng)

        reqBody := GeminiRequest{
                Contents: []GeminiContent{
                        {
                                Parts: []GeminiPart{
                                        {Text: prompt},
                                },
                        },
                },
        }

        jsonData, err := json.Marshal(reqBody)
        if err != nil {
                return AIAnalysis{
                        Pros: []string{},
                        Cons: []string{"Ошибка обработки запроса"},
                }
        }

        url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)
        resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
        if err != nil {
                return AIAnalysis{
                        Pros: []string{},
                        Cons: []string{"Ошибка связи с системой модерации"},
                }
        }
        defer resp.Body.Close()

        body, err := io.ReadAll(resp.Body)
        if err != nil {
                return AIAnalysis{
                        Pros: []string{},
                        Cons: []string{"Ошибка чтения ответа системы"},
                }
        }

        var geminiResp GeminiResponse
        if err := json.Unmarshal(body, &geminiResp); err != nil {
                return AIAnalysis{
                        Pros: []string{},
                        Cons: []string{"Ошибка обработки ответа системы"},
                }
        }

        if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
                return AIAnalysis{
                        Pros: []string{},
                        Cons: []string{"Не удалось получить анализ проекта"},
                }
        }

        aiResponse := geminiResp.Candidates[0].Content.Parts[0].Text
        aiResponse = strings.TrimSpace(aiResponse)
        aiResponse = strings.Trim(aiResponse, "`")
        aiResponse = strings.TrimPrefix(aiResponse, "json")
        aiResponse = strings.TrimSpace(aiResponse)

        var result struct {
                Pros []string `json:"pros"`
                Cons []string `json:"cons"`
        }

        if err := json.Unmarshal([]byte(aiResponse), &result); err != nil {
                return AIAnalysis{
                        Pros: []string{},
                        Cons: []string{"Ошибка интерпретации ответа AI"},
                }
        }

        return AIAnalysis{
                Pros: result.Pros,
                Cons: result.Cons,
        }
}

func ValidateVoteCommentWithGemini(comment string) (bool, string) {
        apiKey := os.Getenv("GEMINI_API_KEY")
        if apiKey == "" {
                return false, "Ошибка конфигурации системы модерации"
        }

        prompt := fmt.Sprintf(`Ты эксперт по партисипаторному бюджетированию. Оцени комментарий голосующего.

КОММЕНТАРИЙ: %s

КРИТЕРИИ ОТКЛОНЕНИЯ:
1. Короче 200 символов
2. Не содержит обоснования (просто "Нравится", "Круто", "Топ")
3. Содержит ненормативную лексику или токсичность
4. Не относится к городской пользе проекта
5. Не объясняет, ПОЧЕМУ человек поддерживает проект

КРИТЕРИИ ОДОБРЕНИЯ:
- Минимум 200 символов
- Есть объяснение, почему проект важен
- Упоминается польза для города или жителей
- Конструктивный и обоснованный комментарий
- Нет токсичности

Ответь СТРОГО в формате JSON:
{
  "approved": true/false,
  "reason": "краткое объяснение на русском языке (одно предложение)"
}`, comment)

        reqBody := GeminiRequest{
                Contents: []GeminiContent{
                        {
                                Parts: []GeminiPart{
                                        {Text: prompt},
                                },
                        },
                },
        }

        jsonData, err := json.Marshal(reqBody)
        if err != nil {
                return false, "Ошибка обработки запроса"
        }

        url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)
        resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
        if err != nil {
                return false, "Ошибка связи с системой модерации"
        }
        defer resp.Body.Close()

        body, err := io.ReadAll(resp.Body)
        if err != nil {
                return false, "Ошибка чтения ответа системы"
        }

        var geminiResp GeminiResponse
        if err := json.Unmarshal(body, &geminiResp); err != nil {
                return false, "Ошибка обработки ответа системы"
        }

        if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
                return false, "Не удалось получить оценку комментария"
        }

        aiResponse := geminiResp.Candidates[0].Content.Parts[0].Text
        aiResponse = strings.TrimSpace(aiResponse)
        aiResponse = strings.Trim(aiResponse, "`")
        aiResponse = strings.TrimPrefix(aiResponse, "json")
        aiResponse = strings.TrimSpace(aiResponse)

        var result struct {
                Approved bool   `json:"approved"`
                Reason   string `json:"reason"`
        }

        if err := json.Unmarshal([]byte(aiResponse), &result); err != nil {
                return false, "Ошибка интерпретации ответа AI"
        }

        return result.Approved, result.Reason
}
