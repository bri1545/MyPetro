package main

import (
        "fmt"
        "log"
        "petropavlovsk-budget/internal/auth"
        "petropavlovsk-budget/internal/db"
)

func main() {
        database, err := db.New()
        if err != nil {
                log.Fatalf("Failed to connect to database: %v", err)
        }
        defer database.Close()

        email := "admin@petro.kz"
        nickname := "Администратор"
        password := "Admin2024!"

        hash, err := auth.HashPassword(password)
        if err != nil {
                log.Fatalf("Failed to hash password: %v", err)
        }

        admin, err := database.CreateAdmin(email, nickname, hash)
        if err != nil {
                log.Fatalf("Failed to create admin: %v", err)
        }

        fmt.Println("✅ Администратор создан успешно!")
        fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        fmt.Printf("📧 Email:    %s\n", email)
        fmt.Printf("👤 Никнейм:  %s\n", nickname)
        fmt.Printf("🔑 Пароль:   %s\n", password)
        fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        fmt.Printf("ID: %d, Роль: %s\n", admin.ID, admin.Role)
        fmt.Println("\n⚠️  ВАЖНО: Сохраните эти данные в безопасном месте!")
}
