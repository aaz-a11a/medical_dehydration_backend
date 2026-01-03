package main

import (
	"fmt"
	"log"

	"dehydrotationlab3/internal/app/ds"
	"dehydrotationlab3/internal/app/dsn"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// Полная очистка и сброс последовательности id
	db.Exec("TRUNCATE TABLE symptoms RESTART IDENTITY CASCADE")

	// Создаем ровно 5 симптомов (пятая карточка — Жажда)
	symptoms := []ds.Symptom{
		{
			Title:        "Диурез",
			Category:     "Объективные признаки",
			Description:  "Снижение объема мочи, концентрированная моча",
			Severity:     "Средняя (3-6%)",
			WeightLoss:   "3-6% массы тела",
			FluidNeed:    "50-70 мл/кг",
			RecoveryTime: "6-12 часов",
			ImageURL:     "5.png",
			IsActive:     true,
		},
		{
			Title:        "Эластичность кожи",
			Category:     "Объективные признаки",
			Description:  "Снижение тургора кожи — кожная складка расправляется медленно",
			Severity:     "Средняя (3-6%)",
			WeightLoss:   "3-6% массы тела",
			FluidNeed:    "50-70 мл/кг",
			RecoveryTime: "6-12 часов",
			ImageURL:     "2.png",
			IsActive:     true,
		},
		{
			Title:        "Судороги",
			Category:     "Тяжелые признаки",
			Description:  "Непроизвольные болезненные сокращения мышц при потере электролитов",
			Severity:     "Тяжелая (7-9%)",
			WeightLoss:   "7-9% массы тела",
			FluidNeed:    "70-100 мл/кг",
			RecoveryTime: "12-24 часа",
			ImageURL:     "3.png",
			IsActive:     true,
		},
		{
			Title:        "Состояние глазных яблок",
			Category:     "Объективные признаки",
			Description:  "Глаза выглядят запавшими, с темными кругами, снижение слезоотделения",
			Severity:     "Тяжелая (7-9%)",
			WeightLoss:   "7-9% массы тела",
			FluidNeed:    "70-100 мл/кг",
			RecoveryTime: "12-24 часа",
			ImageURL:     "4.png",
			IsActive:     true,
		},
		{
			Title:        "Жажда",
			Category:     "Ранние признаки",
			Description:  "Неутолимое желание пить воду, часто сопровождающееся сухостью во рту",
			Severity:     "Легкая (1-2%)",
			WeightLoss:   "1-2% массы тела",
			FluidNeed:    "30-50 мл/кг",
			RecoveryTime: "2-4 часа",
			ImageURL:     "1.png",
			IsActive:     true,
		},
	}

	for _, symptom := range symptoms {
		db.Create(&symptom)
		fmt.Printf("Создан симптом: %s\n", symptom.Title)
	}

	fmt.Println("\n Заполнение данных завершено!")
	fmt.Println("Создано ровно 5 симптомов")
	fmt.Println("Приложение: http://localhost:8080")
}
