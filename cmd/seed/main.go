package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"file_db/internal/engine"
	"file_db/internal/schema"
	"file_db/internal/service"
	"file_db/internal/storage"
)

const dataDir = "data"

func main() {
	reset := flag.Bool("reset", false, "удалить data/ перед заполнением")
	flag.Parse()

	if *reset {
		if err := os.RemoveAll(dataDir); err != nil {
			log.Fatalf("reset: %v", err)
		}
		fmt.Println("data/ удалена")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	svc := mustService()
	seed(svc)

	for _, eng := range []*engine.Engine{
		svc.Seekers, svc.Employers, svc.Moderators,
		svc.Resumes, svc.Vacancies, svc.Skills,
		svc.ResumeSkills, svc.VacancySkills,
		svc.Applications, svc.Messages, svc.Reviews,
	} {
		if err := eng.Close(); err != nil {
			log.Printf("close: %v", err)
		}
	}
	fmt.Println("готово")
}

func mustService() *service.Service {
	must := func(s *schema.Schema, err error) *schema.Schema {
		if err != nil {
			log.Fatalf("schema: %v", err)
		}
		return s
	}
	e := &service.Engines{
		Seekers:       openEng("seeker", must(schema.SeekerSchema())),
		Employers:     openEng("employer", must(schema.EmployerSchema())),
		Moderators:    openEng("moderator", must(schema.ModeratorSchema())),
		Resumes:       openEng("resume", must(schema.ResumeSchema())),
		Vacancies:     openEng("vacancy", must(schema.VacancySchema())),
		Skills:        openEng("skill", must(schema.SkillSchema())),
		ResumeSkills:  openEng("resume_skill", must(schema.ResumeSkillSchema())),
		VacancySkills: openEng("vacancy_skill", must(schema.VacancySkillSchema())),
		Applications:  openEng("application", must(schema.ApplicationSchema())),
		Messages:      openEng("message", must(schema.MessageSchema())),
		Reviews:       openEng("review", must(schema.ReviewSchema())),
	}
	svc, err := service.New(e)
	if err != nil {
		log.Fatalf("service: %v", err)
	}
	return svc
}

func openEng(name string, s *schema.Schema) *engine.Engine {
	path := filepath.Join(dataDir, name+".db")
	db, err := storage.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		db, err = storage.Create(path, s)
	}
	if err != nil {
		log.Fatalf("open %s: %v", name, err)
	}
	eng, err := engine.New(db, s, name, path)
	if err != nil {
		log.Fatalf("engine %s: %v", name, err)
	}
	return eng
}

func check(step string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", step, err)
	}
}

func seed(s *service.Service) {
	moderators := []struct{ email, name string }{
		{"ekaterina.volkova@hh.local", "Екатерина Волкова"},
		{"alexey.sokolov@hh.local", "Алексей Соколов"},
		{"marina.lebedeva@hh.local", "Марина Лебедева"},
		{"dmitry.kozlov@hh.local", "Дмитрий Козлов"},
		{"olga.novikova@hh.local", "Ольга Новикова"},
		{"sergey.morozov@hh.local", "Сергей Морозов"},
	}
	for _, m := range moderators {
		check("moderator", s.AddModerator(map[string]any{
			"email":     m.email,
			"full_name": m.name,
		}))
	}

	seekers := []struct{ email, name, city string }{
		{"anna.ivanova@mail.ru", "Анна Иванова", "Москва"},
		{"boris.petrov@mail.ru", "Борис Петров", "Санкт-Петербург"},
		{"ksenia.sidorova@mail.ru", "Ксения Сидорова", "Казань"},
		{"dmitry.kuznetsov@mail.ru", "Дмитрий Кузнецов", "Москва"},
		{"elena.smirnova@mail.ru", "Елена Смирнова", "Новосибирск"},
		{"fedor.volkov@mail.ru", "Фёдор Волков", "Казань"},
		{"galina.morozova@mail.ru", "Галина Морозова", "Москва"},
		{"gennady.novikov@mail.ru", "Геннадий Новиков", "Екатеринбург"},
		{"igor.sokolov@mail.ru", "Игорь Соколов", "Санкт-Петербург"},
		{"yulia.popova@mail.ru", "Юлия Попова", "Москва"},
		{"maxim.lebedev@mail.ru", "Максим Лебедев", "Нижний Новгород"},
		{"olga.kozlova@mail.ru", "Ольга Козлова", "Казань"},
		{"sergey.novikov@yandex.ru", "Сергей Новиков", "Москва"},
		{"tatiana.morozova@yandex.ru", "Татьяна Морозова", "Санкт-Петербург"},
		{"andrey.volkov@yandex.ru", "Андрей Волков", "Екатеринбург"},
		{"natalia.sokolova@yandex.ru", "Наталья Соколова", "Москва"},
		{"pavel.lebedev@yandex.ru", "Павел Лебедев", "Новосибирск"},
		{"irina.kuznetsova@yandex.ru", "Ирина Кузнецова", "Казань"},
		{"roman.smirnov@yandex.ru", "Роман Смирнов", "Москва"},
		{"svetlana.popova@yandex.ru", "Светлана Попова", "Санкт-Петербург"},
		{"nikita.ivanov@yandex.ru", "Никита Иванов", "Нижний Новгород"},
		{"maria.smirnova@gmail.com", "Мария Смирнова", "Москва"},
		{"artem.kozlov@gmail.com", "Артём Козлов", "Самара"},
		{"evgenia.volkova@gmail.com", "Евгения Волкова", "Москва"},
		{"vladimir.sokolov@gmail.com", "Владимир Соколов", "Санкт-Петербург"},
		{"darya.lebedeva@gmail.com", "Дарья Лебедева", "Казань"},
		{"alexander.morozov@gmail.com", "Александр Морозов", "Москва"},
		{"ekaterina.novikova@gmail.com", "Екатерина Новикова", "Екатеринбург"},
		{"mikhail.petrov@gmail.com", "Михаил Петров", "Новосибирск"},
		{"oksana.sidorova@gmail.com", "Оксана Сидорова", "Москва"},
		{"ivan.kuznetsov@inbox.ru", "Иван Кузнецов", "Санкт-Петербург"},
		{"larisa.ivanova@inbox.ru", "Лариса Иванова", "Казань"},
		{"kirill.smirnov@inbox.ru", "Кирилл Смирнов", "Москва"},
		{"valentina.popova@inbox.ru", "Валентина Попова", "Самара"},
		{"denis.volkov@inbox.ru", "Денис Волков", "Нижний Новгород"},
		{"polina.sokolova@inbox.ru", "Полина Соколова", "Москва"},
		{"vitaly.lebedev@inbox.ru", "Виталий Лебедев", "Санкт-Петербург"},
		{"margarita.kozlova@inbox.ru", "Маргарита Козлова", "Казань"},
		{"oleg.morozov@inbox.ru", "Олег Морозов", "Екатеринбург"},
		{"veronika.novikova@list.ru", "Вероника Новикова", "Москва"},
		{"anton.petrov@list.ru", "Антон Петров", "Новосибирск"},
		{"lyudmila.sidorova@list.ru", "Людмила Сидорова", "Санкт-Петербург"},
		{"georgy.kuznetsov@list.ru", "Георгий Кузнецов", "Москва"},
		{"alina.smirnova@list.ru", "Алина Смирнова", "Казань"},
		{"valery.volkov@list.ru", "Валерий Волков", "Самара"},
		{"kristina.sokolova@list.ru", "Кристина Соколова", "Москва"},
		{"stanislav.lebedev@list.ru", "Станислав Лебедев", "Нижний Новгород"},
		{"inna.kozlova@bk.ru", "Инна Козлова", "Санкт-Петербург"},
		{"eduard.morozov@bk.ru", "Эдуард Морозов", "Москва"},
		{"victoria.novikova@bk.ru", "Виктория Новикова", "Екатеринбург"},
	}
	for _, sk := range seekers {
		check("seeker", s.AddSeeker(map[string]any{
			"email":     sk.email,
			"full_name": sk.name,
			"city":      sk.city,
			"status":    "active",
		}))
	}

	employers := []struct{ email, company, city string }{
		{"hr@yandex.ru", "Яндекс", "Москва"},
		{"hr@vk.com", "ВКонтакте", "Санкт-Петербург"},
		{"hr@tbank.ru", "Т-Банк", "Москва"},
		{"hr@ozon.ru", "Ozon", "Москва"},
		{"hr@avito.ru", "Avito", "Москва"},
		{"hr@kaspersky.com", "Лаборатория Касперского", "Москва"},
		{"hr@wildberries.ru", "Wildberries", "Москва"},
		{"hr@sber.ru", "Сбер", "Москва"},
		{"hr@alfabank.ru", "Альфа-Банк", "Москва"},
		{"hr@vktech.ru", "VK Tech", "Санкт-Петербург"},
		{"hr@ptsecurity.com", "Positive Technologies", "Москва"},
		{"hr@1c.ru", "1С", "Москва"},
		{"hr@skbkontur.ru", "СКБ Контур", "Екатеринбург"},
		{"hr@mts.ru", "МТС Digital", "Москва"},
		{"hr@selectel.ru", "Selectel", "Санкт-Петербург"},
		{"hr@yandexcloud.ru", "Yandex Cloud", "Москва"},
		{"hr@sbertech.ru", "Сбертех", "Москва"},
		{"hr@x5.ru", "X5 Tech", "Москва"},
		{"hr@tochka.com", "Точка Банк", "Екатеринбург"},
		{"hr@ozontech.ru", "Ozon Tech", "Москва"},
		{"hr@vkplay.ru", "VK Play", "Санкт-Петербург"},
		{"hr@dodopizza.com", "Dodo Engineering", "Москва"},
		{"hr@skyeng.ru", "Skyeng", "Москва"},
		{"hr@youdo.com", "YouDo", "Москва"},
		{"hr@megafon.ru", "Мегафон", "Москва"},
	}
	for _, em := range employers {
		check("employer", s.AddEmployer(map[string]any{
			"email":               em.email,
			"company_name":        em.company,
			"city":                em.city,
			"verification_status": "pending",
		}))
	}

	skills := []string{
		"Go", "Python", "Java", "C++", "Rust", "C#", "Kotlin", "Swift", "Scala", "PHP", "Ruby", "Elixir",
		"PostgreSQL", "MySQL", "MongoDB", "Redis", "Cassandra", "ClickHouse", "Oracle", "MS SQL", "Elasticsearch",
		"Docker", "Kubernetes", "Terraform", "Ansible", "GitLab CI", "Jenkins", "GitHub Actions", "Prometheus", "Grafana", "Nginx",
		"React", "Vue", "Angular", "Svelte", "Next.js", "Node.js", "Express", "Django", "FastAPI", "Spring Boot", "Kafka", "RabbitMQ",
		"Spark", "Airflow", "Hadoop", "Git", "Linux", "REST API", "GraphQL", "gRPC", "Microservices", "SQL", "TDD", "Agile", "Machine Learning", "Data Science",
	}
	for _, name := range skills {
		check("skill", s.AddSkill(name))
	}
	skillID := make(map[string]int64, len(skills))
	for i, name := range skills {
		skillID[name] = int64(i + 1)
	}

	resumes := []struct {
		seekerID   int64
		position   string
		salary     int64
		category   string
		employment string
		schedule   string
		status     string
	}{
		{1, "Go-разработчик", 200000, "IT", "full", "remote", "active"},
		{1, "Backend Engineer", 180000, "IT", "full", "office", "active"},
		{2, "Python Developer", 180000, "IT", "full", "hybrid", "active"},
		{3, "Frontend Developer", 140000, "IT", "full", "remote", "active"},
		{4, "DevOps Engineer", 240000, "IT", "full", "remote", "active"},
		{5, "Data Scientist", 260000, "IT", "full", "hybrid", "active"},
		{6, "QA Engineer", 130000, "IT", "part", "remote", "active"},
		{6, "QA Automation", 150000, "IT", "full", "remote", "active"},
		{7, "Системный аналитик", 180000, "IT", "full", "office", "active"},
		{8, "Fullstack разработчик", 200000, "IT", "full", "remote", "active"},
		{9, "Senior Go Developer", 320000, "IT", "full", "remote", "active"},
		{10, "Product Manager", 220000, "Management", "full", "hybrid", "active"},
		{11, "Java Developer", 200000, "IT", "full", "remote", "active"},
		{12, "C++ Developer", 230000, "IT", "full", "office", "active"},
		{13, "Rust Developer", 280000, "IT", "full", "remote", "active"},
		{14, "Business Analyst", 170000, "Management", "full", "hybrid", "active"},
		{15, "Node.js Developer", 190000, "IT", "full", "remote", "active"},
		{16, "QA Lead", 250000, "IT", "full", "remote", "active"},
		{17, "ML Engineer", 300000, "IT", "full", "hybrid", "active"},
		{18, "Frontend (React)", 160000, "IT", "full", "remote", "active"},
		{19, "iOS Developer", 220000, "IT", "full", "remote", "active"},
		{20, "Android Developer", 210000, "IT", "full", "remote", "active"},
		{21, "Python Developer", 170000, "IT", "full", "hybrid", "active"},
		{22, "Data Analyst", 150000, "IT", "full", "remote", "active"},
		{23, "DevOps Engineer", 260000, "IT", "full", "remote", "active"},
		{24, "UX/UI Designer", 140000, "Design", "full", "remote", "active"},
		{25, "C# Developer", 200000, "IT", "full", "office", "active"},
		{25, ".NET Developer", 190000, "IT", "full", "remote", "active"},
		{26, "Vue.js Developer", 150000, "IT", "full", "remote", "active"},
		{27, "Go Developer", 250000, "IT", "full", "remote", "active"},
		{28, "HR Manager", 130000, "HR", "full", "office", "active"},
		{29, "Kubernetes Engineer", 280000, "IT", "full", "remote", "active"},
		{30, "Project Manager", 200000, "Management", "full", "hybrid", "active"},
		{31, "Backend (Python)", 180000, "IT", "full", "remote", "active"},
		{32, "Technical Writer", 130000, "IT", "part", "remote", "active"},
		{33, "Backend Developer (Go)", 230000, "IT", "full", "hybrid", "active"},
		{34, "Marketing Manager", 150000, "Marketing", "full", "hybrid", "active"},
		{35, "Site Reliability Engineer", 300000, "IT", "full", "remote", "active"},
		{36, "Data Engineer", 220000, "IT", "full", "remote", "active"},
		{37, "Java Backend Developer", 240000, "IT", "full", "hybrid", "active"},
		{38, "Frontend Developer", 180000, "IT", "full", "remote", "active"},
		{39, "C++ Developer", 250000, "IT", "full", "office", "active"},
		{40, "Product Analyst", 170000, "IT", "full", "remote", "active"},
		{41, "Go Backend Developer", 260000, "IT", "full", "remote", "active"},
		{42, "QA Engineer", 120000, "IT", "full", "hybrid", "active"},
		{43, "Python Backend Developer", 200000, "IT", "full", "remote", "active"},
		{44, "React Developer", 160000, "IT", "full", "remote", "active"},
		{45, "DevOps", 250000, "IT", "full", "remote", "active"},
		{46, "Data Scientist", 220000, "IT", "full", "hybrid", "active"},
		{47, "Team Lead", 380000, "IT", "full", "remote", "active"},
		{48, "Системный администратор", 140000, "IT", "full", "office", "active"},
		{49, "Backend Engineer", 270000, "IT", "full", "remote", "active"},
		{50, "Product Designer", 170000, "Design", "full", "remote", "active"},
	}
	for _, r := range resumes {
		check("resume", s.AddResume(map[string]any{
			"seeker_id":       r.seekerID,
			"position":        r.position,
			"desired_salary":  r.salary,
			"category":        r.category,
			"employment_type": r.employment,
			"schedule":        r.schedule,
			"status":          r.status,
		}))
	}

	type vacancySeed struct {
		employerID int64
		title      string
		city       string
		salaryFrom int64
		salaryTo   int64
		category   string
		employment string
		schedule   string
		status     string
	}
	vacancies := []vacancySeed{
		// Яндекс
		{1, "Go-разработчик в поиск", "Москва", 250000, 350000, "IT", "full", "hybrid", "active"},
		{1, "Senior Backend (Go)", "Москва", 300000, 450000, "IT", "full", "remote", "active"},
		{1, "Frontend Developer (React)", "Москва", 200000, 300000, "IT", "full", "hybrid", "active"},
		{1, "ML Engineer", "Москва", 280000, 420000, "IT", "full", "remote", "active"},
		{1, "Site Reliability Engineer", "Москва", 280000, 380000, "IT", "full", "remote", "active"},
		// ВКонтакте
		{2, "Backend Developer (Go)", "Санкт-Петербург", 220000, 320000, "IT", "full", "remote", "active"},
		{2, "Frontend Developer (React)", "Санкт-Петербург", 200000, 300000, "IT", "full", "hybrid", "active"},
		{2, "iOS Developer", "Санкт-Петербург", 240000, 340000, "IT", "full", "remote", "active"},
		{2, "Product Manager", "Санкт-Петербург", 250000, 400000, "Management", "full", "hybrid", "active"},
		// Т-Банк
		{3, "Java-разработчик", "Москва", 240000, 340000, "IT", "full", "office", "active"},
		{3, "SRE / DevOps", "Москва", 300000, 420000, "IT", "full", "remote", "active"},
		{3, "Python Developer", "Москва", 200000, 280000, "IT", "full", "remote", "active"},
		{3, "Data Analyst", "Москва", 180000, 260000, "IT", "full", "hybrid", "active"},
		// Ozon
		{4, "Go-разработчик", "Москва", 230000, 330000, "IT", "full", "remote", "active"},
		{4, "Rust Developer", "Москва", 300000, 450000, "IT", "full", "remote", "active"},
		{4, "QA Automation", "Москва", 160000, 220000, "IT", "full", "remote", "pending"},
		{4, "Аналитик данных", "Москва", 180000, 260000, "IT", "full", "hybrid", "active"},
		// Avito
		{5, "Backend Developer (Go)", "Москва", 250000, 350000, "IT", "full", "hybrid", "active"},
		{5, "Fullstack (Go + React)", "Москва", 230000, 330000, "IT", "full", "remote", "active"},
		{5, "DevOps Engineer", "Москва", 250000, 350000, "IT", "full", "remote", "active"},
		// Касперский
		{6, "C++ Developer", "Москва", 250000, 350000, "IT", "full", "office", "active"},
		{6, "Python Developer", "Москва", 200000, 280000, "IT", "full", "hybrid", "active"},
		{6, "QA Engineer", "Москва", 140000, 200000, "IT", "full", "remote", "active"},
		// Wildberries
		{7, "Go-разработчик", "Москва", 220000, 320000, "IT", "full", "remote", "active"},
		{7, "Backend Developer (Java)", "Москва", 220000, 300000, "IT", "full", "remote", "active"},
		{7, "React Developer", "Москва", 180000, 260000, "IT", "full", "hybrid", "active"},
		// Сбер
		{8, "Java Backend Developer", "Москва", 250000, 350000, "IT", "full", "office", "active"},
		{8, "Python Developer", "Москва", 200000, 280000, "IT", "full", "hybrid", "active"},
		{8, "Senior QA", "Москва", 180000, 250000, "IT", "full", "remote", "active"},
		{8, "Kotlin Developer", "Москва", 260000, 360000, "IT", "full", "remote", "active"},
		// Альфа-Банк
		{9, "Java Developer", "Москва", 240000, 340000, "IT", "full", "hybrid", "active"},
		{9, "Go Developer", "Москва", 260000, 360000, "IT", "full", "remote", "active"},
		{9, "Product Manager", "Москва", 250000, 400000, "Management", "full", "hybrid", "active"},
		// VK Tech
		{10, "Kotlin Developer", "Санкт-Петербург", 250000, 350000, "IT", "full", "remote", "active"},
		{10, "Backend (Go)", "Санкт-Петербург", 240000, 340000, "IT", "full", "hybrid", "active"},
		{10, "SRE", "Санкт-Петербург", 280000, 400000, "IT", "full", "remote", "active"},
		// Positive Technologies
		{11, "Python Developer", "Москва", 200000, 300000, "IT", "full", "hybrid", "active"},
		{11, "Reverse Engineer", "Москва", 280000, 400000, "IT", "full", "office", "active"},
		// 1С
		{12, "C++ Developer", "Москва", 200000, 280000, "IT", "full", "office", "active"},
		{12, "Java Developer", "Москва", 180000, 260000, "IT", "full", "hybrid", "active"},
		// СКБ Контур
		{13, "Backend (Python)", "Екатеринбург", 180000, 260000, "IT", "full", "remote", "active"},
		{13, "Frontend (React)", "Екатеринбург", 160000, 240000, "IT", "full", "hybrid", "active"},
		// МТС Digital
		{14, "Java Developer", "Москва", 220000, 320000, "IT", "full", "remote", "active"},
		{14, "Data Engineer", "Москва", 220000, 300000, "IT", "full", "hybrid", "active"},
		{14, "Go Developer", "Москва", 240000, 340000, "IT", "full", "remote", "active"},
		// Selectel
		{15, "SRE / DevOps", "Санкт-Петербург", 250000, 350000, "IT", "full", "remote", "active"},
		{15, "Go Developer", "Санкт-Петербург", 220000, 320000, "IT", "full", "hybrid", "active"},
		// Yandex Cloud
		{16, "Go Developer", "Москва", 280000, 400000, "IT", "full", "remote", "active"},
		{16, "Kubernetes Engineer", "Москва", 300000, 420000, "IT", "full", "remote", "active"},
		{16, "Technical Writer", "Москва", 150000, 220000, "IT", "full", "hybrid", "active"},
		// Сбертех
		{17, "Python Developer", "Москва", 220000, 320000, "IT", "full", "hybrid", "active"},
		{17, "Java Developer", "Москва", 240000, 340000, "IT", "full", "remote", "active"},
		{17, "Data Scientist", "Москва", 260000, 380000, "IT", "full", "hybrid", "active"},
		// X5 Tech
		{18, "Python Developer", "Москва", 200000, 300000, "IT", "full", "remote", "active"},
		{18, "Go Developer", "Москва", 240000, 340000, "IT", "full", "hybrid", "active"},
		{18, "QA Automation", "Москва", 160000, 230000, "IT", "full", "remote", "active"},
		// Точка Банк
		{19, "Go Developer", "Екатеринбург", 220000, 320000, "IT", "full", "remote", "active"},
		{19, "Backend (Kotlin)", "Екатеринбург", 200000, 300000, "IT", "full", "hybrid", "active"},
		// Ozon Tech
		{20, "ML Engineer", "Москва", 280000, 400000, "IT", "full", "remote", "active"},
		{20, "Backend (Go)", "Москва", 230000, 330000, "IT", "full", "hybrid", "active"},
		// VK Play
		{21, "Frontend Developer (React)", "Санкт-Петербург", 180000, 270000, "IT", "full", "remote", "active"},
		{21, "Backend (Go)", "Санкт-Петербург", 220000, 320000, "IT", "full", "hybrid", "active"},
		// Dodo Engineering
		{22, "Go Developer", "Москва", 230000, 330000, "IT", "full", "remote", "active"},
		{22, "Frontend (Vue)", "Москва", 180000, 260000, "IT", "full", "hybrid", "active"},
		// Skyeng
		{23, "Python Developer", "Москва", 190000, 280000, "IT", "full", "remote", "active"},
		{23, "Frontend Developer", "Москва", 170000, 250000, "IT", "full", "remote", "pending"},
		// YouDo
		{24, "Fullstack Developer", "Москва", 180000, 270000, "IT", "full", "remote", "pending"},
		// Мегафон
		{25, "Java Developer", "Москва", 220000, 320000, "IT", "full", "hybrid", "active"},
		{25, "Data Engineer", "Москва", 230000, 320000, "IT", "full", "remote", "active"},
	}
	for i, v := range vacancies {
		modID := int64(0)
		if v.status == "active" {
			modID = int64(1 + i%len(moderators))
		}
		check("vacancy", s.AddVacancy(map[string]any{
			"employer_id":     v.employerID,
			"title":           v.title,
			"city":            v.city,
			"salary_from":     v.salaryFrom,
			"salary_to":       v.salaryTo,
			"category":        v.category,
			"employment_type": v.employment,
			"schedule":        v.schedule,
			"status":          v.status,
			"moderator_id":    modID,
		}))
	}

	rng := rand.New(rand.NewSource(42))
	addSkills := func(label string, add func(any, any) error, ownerID int64, relevant []string, n int) {
		if n > len(relevant) {
			n = len(relevant)
		}
		perm := rng.Perm(len(relevant))
		for i := 0; i < n; i++ {
			name := relevant[perm[i]]
			if id, ok := skillID[name]; ok {
				check(label, add(ownerID, id))
			}
		}
	}
	for i, r := range resumes {
		addSkills("resume_skill", s.AddResumeSkill, int64(i+1), skillsFor(r.position), 3+rng.Intn(2))
	}
	for i, v := range vacancies {
		addSkills("vacancy_skill", s.AddVacancySkill, int64(i+1), skillsFor(v.title), 2+rng.Intn(3))
	}

	type app struct{ seekerID, resumeID, vacancyID int64 }
	apps := []app{
		// Go
		{1, 1, 1}, {1, 1, 2}, {1, 1, 14}, {1, 2, 18},
		{9, 11, 2}, {9, 11, 48}, {9, 11, 60},
		{27, 30, 14}, {27, 30, 24}, {27, 30, 32},
		{33, 36, 55}, {33, 36, 45},
		{41, 44, 45}, {41, 44, 62}, {41, 44, 63},
		{49, 52, 45}, {49, 52, 60},
		// Python
		{2, 3, 12}, {2, 3, 22}, {2, 3, 51},
		{21, 23, 65}, {21, 23, 41},
		{31, 34, 12}, {31, 34, 51},
		{43, 46, 51}, {43, 46, 28},
		// Frontend
		{3, 4, 3}, {3, 4, 7}, {3, 4, 42},
		{18, 20, 3}, {18, 20, 26},
		{26, 29, 61}, {26, 29, 64},
		{38, 41, 3}, {38, 41, 26}, {38, 41, 61},
		{44, 47, 7}, {44, 47, 26},
		// DevOps / SRE
		{4, 5, 11}, {4, 5, 20}, {4, 5, 36},
		{23, 25, 11}, {23, 25, 46},
		{29, 32, 46}, {29, 32, 49},
		{35, 38, 5}, {35, 38, 11}, {35, 38, 49},
		{45, 48, 20}, {45, 48, 36},
		// Data
		{5, 6, 4}, {5, 6, 53}, {5, 6, 59},
		{17, 19, 4}, {17, 19, 53}, {17, 19, 59},
		{22, 24, 13}, {22, 24, 17},
		{36, 39, 44}, {36, 39, 69},
		{40, 43, 13}, {40, 43, 17},
		{46, 49, 53}, {46, 49, 59},
		// QA
		{6, 7, 23}, {6, 7, 29},
		{6, 8, 16}, {6, 8, 56},
		{16, 18, 29}, {16, 18, 56},
		{42, 45, 23}, {42, 45, 29},
		// Java
		{11, 13, 10}, {11, 13, 27}, {11, 13, 31},
		{37, 40, 10}, {37, 40, 27}, {37, 40, 43},
		{25, 27, 10}, {25, 28, 31},
		// C++
		{12, 14, 21}, {12, 14, 39},
		{39, 42, 21}, {39, 42, 39},
		// Rust
		{13, 15, 15},
		// Mobile
		{19, 21, 8},
		{20, 22, 34}, {20, 22, 58},
		// Management
		{10, 12, 9}, {10, 12, 33},
		{14, 16, 33}, {30, 33, 9}, {30, 33, 33},
		{47, 50, 9},
		// Fullstack
		{8, 10, 19}, {8, 10, 67},
		{15, 17, 19}, {15, 17, 67},
		// Misc
		{32, 35, 50},
	}
	for _, a := range apps {
		err := s.AddApplication(map[string]any{
			"seeker_id":  a.seekerID,
			"resume_id":  a.resumeID,
			"vacancy_id": a.vacancyID,
		})
		if err != nil {
			fmt.Printf("skip app seeker=%d resume=%d vacancy=%d: %v\n",
				a.seekerID, a.resumeID, a.vacancyID, err)
		}
	}

	messages := []struct {
		authorType string
		authorID   int64
		text       string
	}{
		{"seeker", 1, "Здравствуйте! Меня заинтересовала вакансия Go-разработчик."},
		{"employer", 1, "Добрый день! Расскажите, пожалуйста, о вашем опыте с Go."},
		{"seeker", 1, "Пять лет коммерческой разработки, последние три года - микросервисы на Go."},
		{"employer", 1, "Отлично. Когда вам удобно провести техническое собеседование?"},
		{"seeker", 1, "В любой день после 18:00 или в субботу."},
		{"employer", 1, "Записал вас на четверг, 19:00. Ссылка придёт на почту."},
		{"seeker", 1, "Спасибо, буду."},

		{"seeker", 2, "Здравствуйте! Хочу откликнуться на позицию Python Developer."},
		{"employer", 3, "Приветствуем! Пришлите, пожалуйста, актуальное резюме."},
		{"seeker", 2, "Резюме прикреплено в отклике, есть опыт с FastAPI и PostgreSQL."},
		{"employer", 3, "Спасибо, изучим и вернёмся с ответом в течение двух дней."},

		{"seeker", 3, "Добрый день! Интересует вакансия Frontend Developer (React)."},
		{"employer", 2, "Здравствуйте! Какой у вас опыт с React и TypeScript?"},
		{"seeker", 3, "Три года с React, два - с TypeScript, писала дизайн-систему."},
		{"employer", 2, "Звучит интересно. Можете выполнить тестовое задание?"},
		{"seeker", 3, "Да, конечно. Сколько времени на него отводится?"},
		{"employer", 2, "Три дня, задание оплачиваемое."},

		{"seeker", 4, "Здравствуйте! Видел вакансию DevOps Engineer."},
		{"employer", 3, "Добрый! Расскажите об опыте с Kubernetes и Terraform."},
		{"seeker", 4, "Четыре года в продакшене, строил кластеры с нуля."},
		{"employer", 3, "Замечательно, давайте созвонимся на этой неделе."},

		{"seeker", 5, "Здравствуйте! Откликаюсь на вакансию Data Scientist."},
		{"employer", 8, "Добрый день! Есть опыт с рекомендательными системами?"},
		{"seeker", 5, "Да, делала модель для e-commerce с приростом CTR на 12%."},
		{"employer", 8, "Интересно. Какие инструменты использовали?"},
		{"seeker", 5, "Python, CatBoost, Spark, Airflow."},
		{"employer", 8, "Приглашаем на техническое интервью."},

		{"seeker", 6, "Здравствуйте! Интересует позиция QA Engineer."},
		{"employer", 6, "Добрый день! Работали с автотестами на Python?"},
		{"seeker", 6, "Да, pytest + Selenium, есть опыт с Playwright."},
		{"employer", 6, "Отлично, отправим тестовое задание."},

		{"employer", 5, "К сожалению, по итогам собеседования мы выбрали другого кандидата. Спасибо за уделённое время!"},
		{"seeker", 7, "Спасибо за обратную связь и оперативный ответ."},

		{"seeker", 8, "Здравствуйте! Откликнулся на Fullstack-вакансию."},
		{"employer", 5, "Привет! Есть опыт с Go и React в одном проекте?"},
		{"seeker", 8, "Да, делал B2B-платформу на этом стеке."},
		{"employer", 5, "Отлично, приглашаем на техническое интервью."},

		{"seeker", 9, "Здравствуйте! Меня заинтересовала вакансия Senior Backend (Go)."},
		{"employer", 1, "Добрый день! Какой у вас опыт в лидерстве команды?"},
		{"seeker", 9, "Последние два года - техлид команды из шести человек."},
		{"employer", 1, "Хорошо, давайте обсудим детали на созвоне."},

		{"seeker", 10, "Здравствуйте! Интересует позиция Product Manager."},
		{"employer", 2, "Добрый! Расскажите о продуктах, которые вели."},
		{"seeker", 10, "Два года вела B2B SaaS, рост MAU на 40%."},
		{"employer", 2, "Звучит интересно, давайте назначим интервью."},

		{"seeker", 11, "Добрый день! Откликаюсь на вакансию Java-разработчик."},
		{"employer", 3, "Здравствуйте! Spring Boot используете в продакшене?"},
		{"seeker", 11, "Да, последние три проекта - на Spring Boot + Kafka."},
		{"employer", 3, "Отлично, отправим ссылку на техническое интервью."},

		{"seeker", 17, "Здравствуйте! Очень интересует вакансия ML Engineer."},
		{"employer", 1, "Добрый день! С какими моделями работали в продакшене?"},
		{"seeker", 17, "Рекомендательные системы, NLP-модели, ранжирование."},
		{"employer", 1, "Отлично, давайте назначим техническое интервью."},

		{"seeker", 22, "Добрый день! Откликаюсь на позицию Data Analyst."},
		{"employer", 3, "Здравствуйте! Работали с A/B-тестами?"},
		{"seeker", 22, "Да, регулярно, включая дизайн экспериментов."},

		{"seeker", 30, "Здравствуйте! Интересует вакансия Project Manager."},
		{"employer", 2, "Добрый! Какой у вас опыт с Agile-командами?"},
		{"seeker", 30, "Вела Scrum-команды из 8-10 человек в финтехе."},

		{"employer", 6, "К сожалению, вакансия уже закрыта. Будем рады видеть вас в будущем!"},
		{"seeker", 42, "Спасибо, буду следить за вашими вакансиями."},

		{"seeker", 47, "Здравствуйте! Интересует роль Team Lead."},
		{"employer", 1, "Добрый день! Какой у вас управленческий опыт?"},
		{"seeker", 47, "Три года тимлид, до этого - сеньор-разработчик."},
		{"employer", 1, "Приглашаем на интервью с CTO."},
	}
	for _, m := range messages {
		check("message", s.AddMessage(map[string]any{
			"author_type": m.authorType,
			"author_id":   m.authorID,
			"text":        m.text,
		}))
	}

	reviews := []struct {
		seekerID, employerID int64
		rating               int64
		text                 string
		published            bool
	}{
		{1, 1, 5, "Отличный процесс собеседования, всё чётко и по делу.", true},
		{2, 3, 4, "Быстрый фидбэк, вежливые рекрутёры.", true},
		{3, 2, 5, "Профессиональная команда, интересные задачи.", true},
		{4, 3, 3, "Долго не отвечали после финального интервью.", false},
		{5, 8, 4, "Интересные задачи, хорошая команда.", true},
		{6, 6, 2, "Отказ без объяснений после двух недель ожидания.", false},
		{7, 1, 5, "Рекомендую, всё прозрачно.", true},
		{8, 5, 4, "Хороший опыт, техническое интервью по делу.", true},
		{9, 1, 5, "Лучший процесс собеседования, что я видел.", true},
		{10, 2, 4, "Вежливые рекрутёры, чёткие сроки.", true},
		{11, 3, 4, "Собеседование прошло комфортно.", true},
		{12, 6, 3, "Слишком длинный процесс - почти полтора месяца.", true},
		{13, 4, 4, "Хорошая команда, задачи соответствуют описанию.", true},
		{14, 9, 4, "Обратная связь по итогам, всё по существу.", true},
		{15, 5, 5, "Отличный опыт, рекомендую.", true},
		{16, 8, 3, "Тестовое задание было неоправданно объёмным.", true},
		{17, 1, 5, "Очень доволен процессом.", true},
		{18, 2, 4, "Техническое интервью было интересным.", true},
		{19, 2, 4, "Быстрый ответ, понятные следующие шаги.", true},
		{20, 10, 4, "Хорошая организация процесса.", true},
		{21, 3, 3, "Ожидание фидбэка затянулось.", true},
		{22, 3, 4, "Вежливые рекрутёры, комфортное общение.", true},
		{23, 4, 5, "Профессиональные интервьюеры.", true},
		{24, 5, 3, "Не хватило конкретики на техническом интервью.", true},
		{25, 3, 2, "Финальный отказ без обратной связи.", true},
		{26, 2, 4, "Хороший опыт.", true},
		{27, 1, 5, "Очень понравилась команда.", true},
		{28, 8, 3, "Долгий процесс согласования.", true},
		{30, 9, 4, "Быстрая обратная связь, всё понятно.", true},
		{31, 3, 4, "Рекомендую.", true},
		{33, 4, 5, "Отличные задачи и команда.", true},
		{35, 5, 4, "Хорошее техническое интервью.", true},
		{38, 2, 5, "Лучший процесс из всех, где я бывал.", true},
		{40, 5, 4, "Интересно, вежливо, по делу.", true},
		{47, 1, 5, "Профессиональная команда.", true},
	}
	for _, r := range reviews {
		check("review", s.AddReview(map[string]any{
			"seeker_id":    r.seekerID,
			"employer_id":  r.employerID,
			"rating":       r.rating,
			"text":         r.text,
			"is_published": r.published,
		}))
	}

	fmt.Printf("создано: %d модераторов, %d соискателей, %d работодателей, %d навыков, "+
		"%d резюме, %d вакансий, %d откликов, %d сообщений, %d отзывов\n",
		len(moderators), len(seekers), len(employers), len(skills),
		len(resumes), len(vacancies), len(apps), len(messages), len(reviews))
}

func skillsFor(text string) []string {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "go-") || strings.Contains(t, "(go)") || strings.Contains(t, "(go +") ||
		strings.Contains(t, "go developer") || strings.Contains(t, "go backend") || strings.Contains(t, "senior go"):
		return []string{"Go", "PostgreSQL", "Docker", "Kubernetes", "Microservices", "Redis", "gRPC", "Kafka"}
	case strings.Contains(t, "python"):
		return []string{"Python", "FastAPI", "Django", "PostgreSQL", "Docker", "Kafka", "Redis"}
	case strings.Contains(t, "kotlin"):
		return []string{"Kotlin", "Spring Boot", "Kafka", "PostgreSQL", "Docker", "Microservices"}
	case strings.Contains(t, "java"):
		return []string{"Java", "Spring Boot", "Kafka", "PostgreSQL", "Docker", "Kubernetes", "Microservices"}
	case strings.Contains(t, "c++"):
		return []string{"C++", "Linux", "TDD", "Git", "Docker", "PostgreSQL"}
	case strings.Contains(t, "rust"):
		return []string{"Rust", "Linux", "Docker", "PostgreSQL", "gRPC", "Microservices"}
	case strings.Contains(t, "c#") || strings.Contains(t, ".net"):
		return []string{"C#", "MS SQL", "Git", "Docker", "REST API", "Microservices"}
	case strings.Contains(t, "fullstack"):
		return []string{"Go", "React", "TypeScript", "PostgreSQL", "Docker", "REST API"}
	case strings.Contains(t, "vue"):
		return []string{"Vue", "React", "TypeScript", "REST API", "Git", "Next.js"}
	case strings.Contains(t, "react") || strings.Contains(t, "frontend"):
		return []string{"React", "TypeScript", "Next.js", "Vue", "REST API", "Git"}
	case strings.Contains(t, "node"):
		return []string{"Node.js", "TypeScript", "Express", "PostgreSQL", "Redis", "REST API"}
	case strings.Contains(t, "devops") || strings.Contains(t, "sre") ||
		strings.Contains(t, "site reliability") || strings.Contains(t, "kubernetes"):
		return []string{"Docker", "Kubernetes", "Terraform", "Ansible", "Prometheus", "GitLab CI", "Linux"}
	case strings.Contains(t, "ml engineer") || strings.Contains(t, "data scientist") ||
		strings.Contains(t, "machine learning"):
		return []string{"Python", "Machine Learning", "Data Science", "Spark", "PostgreSQL", "Airflow"}
	case strings.Contains(t, "data engineer"):
		return []string{"Python", "Kafka", "Spark", "Airflow", "Hadoop", "PostgreSQL"}
	case strings.Contains(t, "data analyst") || strings.Contains(t, "product analyst") ||
		strings.Contains(t, "аналитик данных"):
		return []string{"SQL", "Python", "ClickHouse", "PostgreSQL", "Data Science"}
	case strings.Contains(t, "analyst"):
		return []string{"SQL", "Python", "ClickHouse", "PostgreSQL"}
	case strings.Contains(t, "qa"):
		return []string{"Python", "TDD", "Git", "Linux", "REST API"}
	case strings.Contains(t, "ios"):
		return []string{"Swift", "Git", "REST API", "Microservices", "Linux"}
	case strings.Contains(t, "android"):
		return []string{"Kotlin", "Git", "REST API", "Microservices", "Linux"}
	case strings.Contains(t, "product manager") || strings.Contains(t, "project manager"):
		return []string{"Agile", "SQL", "REST API", "Microservices"}
	case strings.Contains(t, "аналитик"):
		return []string{"SQL", "Python", "ClickHouse", "PostgreSQL"}
	case strings.Contains(t, "sysadmin") || strings.Contains(t, "системный администратор"):
		return []string{"Linux", "Nginx", "Ansible", "Git", "Docker"}
	case strings.Contains(t, "writer"):
		return []string{"Agile", "Git", "REST API"}
	case strings.Contains(t, "designer"):
		return []string{"Agile", "Git"}
	case strings.Contains(t, "reverse"):
		return []string{"C++", "Linux", "TDD", "Git"}
	case strings.Contains(t, "hr"):
		return []string{"Agile", "Git"}
	case strings.Contains(t, "marketing"):
		return []string{"Agile", "SQL"}
	case strings.Contains(t, "team lead"):
		return []string{"Go", "Microservices", "Kubernetes", "Docker", "PostgreSQL", "Agile"}
	case strings.Contains(t, "backend"):
		return []string{"Go", "PostgreSQL", "Docker", "Microservices", "Redis"}
	default:
		return []string{"Git", "Linux", "SQL"}
	}
}
