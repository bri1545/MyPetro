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

type ModerationResult struct {
        Approved bool
        Reason   string
}

func ValidateIdeaWithGemini(p models.ProjectSubmission) ModerationResult {
        apiKey := os.Getenv("GEMINI_API_KEY")
        if apiKey == "" {
                return ModerationResult{
                        Approved: false,
                        Reason:   "Ошибка конфигурации системы модерации",
                }
        }

        prompt := fmt.Sprintf(`Ты эксперт по партисипаторному бюджетированию для города Петропавловск, Казахстан.

Оцени эту идею проекта и реши: ОДОБРИТЬ или ОТКЛОНИТЬ.

НАЗВАНИЕ: %s
ОПИСАНИЕ: %s
КАТЕГОРИЯ: %s
РАЙОН: %s
БЮДЖЕТ: %d ₸
КООРДИНАТЫ: lat=%f, lng=%f

КРИТЕРИИ ОТКЛОНЕНИЯ:
1. Бюджет меньше 300 000 ₸ или больше 2 000 000 ₸
2. Описание короче 500 символов
3. Координаты отсутствуют (0,0)
4. Содержит ненормативную лексику или токсичность
5. Проект для личного использования (мебель в школе, техника в офис, ремонт кабинета)
6. Нет общественной пользы (проект не улучшает городскую среду для жителей)
7. Отсутствует описание конкретной проблемы и решения
8. Проект не соответствует категории (озеленение, благоустройство, скверы, культура, урбанистика)

КРИТЕРИИ ОДОБРЕНИЯ:
- Проект имеет общественную пользу
- Направлен на улучшение городской среды
- Адекватный бюджет в пределах 300k-2M ₸
- Четко описана проблема и предлагаемое решение
- Подходит под одну из категорий городского развития

Ответь СТРОГО в формате JSON:
{
  "approved": true/false,
  "reason": "краткое объяснение на русском языке (одно предложение)"
}`, p.Title, p.Description, p.Category, p.District, p.Budget, p.Lat, p.Lng)

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
                return ModerationResult{
                        Approved: false,
                        Reason:   "Ошибка обработки запроса",
                }
        }

        url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)
        resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
        if err != nil {
                return ModerationResult{
                        Approved: false,
                        Reason:   "Ошибка связи с системой модерации",
                }
        }
        defer resp.Body.Close()

        body, err := io.ReadAll(resp.Body)
        if err != nil {
                return ModerationResult{
                        Approved: false,
                        Reason:   "Ошибка чтения ответа системы",
                }
        }

        var geminiResp GeminiResponse
        if err := json.Unmarshal(body, &geminiResp); err != nil {
                return ModerationResult{
                        Approved: false,
                        Reason:   "Ошибка обработки ответа системы",
                }
        }

        if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
                return ModerationResult{
                        Approved: false,
                        Reason:   "Не удалось получить оценку проекта",
                }
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
                return ModerationResult{
                        Approved: false,
                        Reason:   "Ошибка интерпретации ответа AI",
                }
        }

        return ModerationResult{
                Approved: result.Approved,
                Reason:   result.Reason,
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
