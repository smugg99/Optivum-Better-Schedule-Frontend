// scraper/scraper.go
package scraper

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"smuggr.xyz/optivum-bsf/common/config"
	"smuggr.xyz/optivum-bsf/common/models"
	"smuggr.xyz/optivum-bsf/common/utils"
	"smuggr.xyz/optivum-bsf/core/hub"
	"smuggr.xyz/optivum-bsf/core/observer"

	"github.com/PuerkitoBio/goquery"
)

var Config config.ScraperConfig

type ScraperResource struct {
	Indexes     []int64
	Designators *models.Designators
	Observer    *observer.Observer
	Hub         *hub.Hub
	IndexRegex  *regexp.Regexp
	Mu          *sync.RWMutex
}

var (
	DivisionIndexRegex = regexp.MustCompile(`o(\d+)\.html`)
	TeacherIndexRegex  = regexp.MustCompile(`n(\d+)\.html`)
	RoomIndexRegex     = regexp.MustCompile(`s(\d+)\.html`)
)

var (
	DivisionsIndexes []int64
	TeachersIndexes  []int64
	RoomsIndexes     []int64

	DivisionsListObserver *observer.Observer
	TeachersListObserver  *observer.Observer
	RoomsListObserver     *observer.Observer

	DivisionsHub *hub.Hub
	TeachersHub  *hub.Hub
	RoomsHub     *hub.Hub

	DivisionsMu sync.RWMutex
	TeachersMu  sync.RWMutex
	RoomsMu     sync.RWMutex
)

var DivisionsDesignators = &models.Designators{
	Designators: make(map[string]int64),
}

var TeachersDesignators = &models.Designators{
	Designators: make(map[string]int64),
}

var RoomsDesignators = &models.Designators{
	Designators: make(map[string]int64),
}

func makeDivisionEndpoint(index int64) string {
	return fmt.Sprintf(Config.Endpoints.Division, index)
}

func makeTeacherEndpoint(index int64) string {
	return fmt.Sprintf(Config.Endpoints.Teacher, index)
}

func makeRoomEndpoint(index int64) string {
	return fmt.Sprintf(Config.Endpoints.Room, index)
}

func splitDivisionTitle(s string) (string, string) {
	parts := strings.Split(s, " ")
	if len(parts) < 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func splitTeacherTitle(s string) (string, string) {
	parts := strings.Split(s, " ")
	if len(parts) < 2 {
		return "", ""
	}

	rawDesignator := strings.TrimSpace(parts[1])
	rawDesignator = strings.Trim(rawDesignator, "()")

	return strings.TrimSpace(parts[0]), rawDesignator
}

func parseTimeRange(s string) (models.TimeRange, error) {
	s = strings.ReplaceAll(s, " ", "")
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return models.TimeRange{}, fmt.Errorf("invalid time range: %s", s)
	}

	_start := strings.TrimSpace(parts[0])
	_end := strings.TrimSpace(parts[1])

	startParts := strings.Split(_start, ":")
	if len(startParts) != 2 {
		return models.TimeRange{}, fmt.Errorf("invalid start time: %s", _start)
	}
	startHour, err := strconv.ParseInt(startParts[0], 10, 32)
	if err != nil {
		return models.TimeRange{}, fmt.Errorf("invalid start hour: %s %v", startParts[0], err)
	}
	startMinute, err := strconv.ParseInt(startParts[1], 10, 32)
	if err != nil {
		return models.TimeRange{}, fmt.Errorf("invalid start minute: %s %v", startParts[1], err)
	}

	endParts := strings.Split(_end, ":")
	if len(endParts) != 2 {
		return models.TimeRange{}, fmt.Errorf("invalid end time: %s %v", _end, err)
	}
	endHour, err := strconv.ParseInt(endParts[0], 10, 32)
	if err != nil {
		return models.TimeRange{}, fmt.Errorf("invalid end hour: %s %v", endParts[0], err)
	}

	endMinute, err := strconv.ParseInt(endParts[1], 10, 32)
	if err != nil {
		return models.TimeRange{}, fmt.Errorf("invalid end minute: %s %v", endParts[1], err)
	}

	start := models.Timestamp{
		Hour:   startHour,
		Minute: startMinute,
	}

	end := models.Timestamp{
		Hour:   endHour,
		Minute: endMinute,
	}

	return models.TimeRange{Start: &start, End: &end}, nil
}

func parseLesson(rowElement *goquery.Selection, timeRange *models.TimeRange) (*models.Lesson, error) {
	lessonName := rowElement.Find("span.p").First().Text()
	// Some lessons contain only the table data with embedded text only
	if lessonName == "" {
		lessonName = rowElement.Text()
	}

	division := rowElement.Find("a.o").First().Text()
	teacher := rowElement.Find("a.n").First().Text()
	room := rowElement.Find("a.s").First().Text()
	lesson := models.Lesson{
		TimeRange:          timeRange,
		FullName:           strings.TrimSpace(lessonName),
		TeacherDesignator:  teacher,
		DivisionDesignator: division,
		RoomDesignator:     room,
	}

	return &lesson, nil
}

func parseLessons(rowElement *goquery.Selection, timeRange *models.TimeRange) ([]*models.Lesson, error) {
	lessons := []*models.Lesson{}
	innerSpanElements := rowElement.Find("span > span.p")
	if innerSpanElements.Length() > 1 {
		innerSpanElements.Each(func(i int, s *goquery.Selection) {
			parentSelection := s.Parent()
			lesson, err := parseLesson(parentSelection, timeRange)
			if err != nil {
				fmt.Println("error parsing lesson", err)
				return
			}
			lessons = append(lessons, lesson)
		})
	} else {
		lesson, err := parseLesson(rowElement, timeRange)
		if err != nil {
			fmt.Println("error parsing lesson", err)
			return nil, err
		}
		lessons = append(lessons, lesson)
	}

	return lessons, nil
}

func scrapeDivisionTitle(doc *goquery.Document) (string, string, error) {
	titleSelection := doc.Find("span.tytulnapis").First()
	if titleSelection.Length() == 0 {
		return "", "", fmt.Errorf("no division title found")
	}

	title := titleSelection.Text()
	designator, fullName := splitDivisionTitle(title)

	return designator, fullName, nil
}

func scrapeTeacherTitle(doc *goquery.Document) (string, string, error) {
	titleSelection := doc.Find("span.tytulnapis").First()
	if titleSelection.Length() == 0 {
		return "", "", fmt.Errorf("no teacher title found")
	}

	title := titleSelection.Text()
	fullName, designator := splitTeacherTitle(title)

	return designator, fullName, nil
}

func scrapeRoomTitle(doc *goquery.Document) (string, error) {
	titleSelection := doc.Find("span.tytulnapis").First()
	if titleSelection.Length() == 0 {
		return "", fmt.Errorf("no room title found")
	}

	title := titleSelection.Text()
	return title, nil
}

func scrapeSchedule(doc *goquery.Document) (*models.Schedule, error) {
	var schedule *models.Schedule
	var timeRange *models.TimeRange

	rowsSelection := doc.Find("table.tabela > tbody > tr")
	firstRow := rowsSelection.First()
	if firstRow.Length() == 0 {
		return schedule, fmt.Errorf("no rows found")
	}

	columnNumber := 0
	dayOfWeek := 0
	columnsCount := firstRow.Children().Length()

	rowsLength := doc.Find("table.tabela > tbody > tr > td.nr").Length() + 1
	lessonsLength := doc.Find("table.tabela > tbody > tr > td.l").Length()

	// First row is the table headers row so it doesn't count
	scheduleStartColumn := columnsCount - (lessonsLength / (rowsLength - 1))
	workDays := columnsCount - scheduleStartColumn

	schedule = &models.Schedule{
		ScheduleDays: make([]*models.ScheduleDay, workDays),
	}

	for i := 0; i < workDays; i++ {
		schedule.ScheduleDays[i] = &models.ScheduleDay{
			LessonGroups: []*models.LessonGroup{},
		}
	}

	doc.Find("table.tabela > tbody > tr > td").Each(func(i int, rowElement *goquery.Selection) {
		if columnNumber >= columnsCount {
			columnNumber = 1
		} else {
			columnNumber++
		}

		// first column is row count
		// second column is time range
		if columnNumber == 2 {
			_timerange, err := parseTimeRange(rowElement.Text())
			if err != nil {
				fmt.Println("error parsing time range", err)
				return
			}
			timeRange = &_timerange
			// other columns are lessons
		} else if columnNumber > scheduleStartColumn {
			if dayOfWeek < workDays {
				dayOfWeek++
			} else {
				dayOfWeek = 1
			}

			if utils.IsEmptyOrInvisible(rowElement.Text()) {
				return
			}
			lessons, err := parseLessons(rowElement, timeRange)
			if err != nil {
				fmt.Println("error parsing lessons", err)
				return
			}
			//fmt.Println("dayOfWeek:", dayOfWeek, "lessons:", lessons, workDays)

			lessonGroup := &models.LessonGroup{
				Lessons: lessons,
			}

			schedule.ScheduleDays[dayOfWeek-1].LessonGroups = append(
				schedule.ScheduleDays[dayOfWeek-1].LessonGroups,
				lessonGroup,
			)
		}
	})

	return schedule, nil
}

func ScrapeDivision(index int64) (*models.Division, error) {
	endpoint := makeDivisionEndpoint(index)
	doc, err := utils.OpenDoc(Config.BaseUrl, endpoint)
	if err != nil {
		return nil, fmt.Errorf("error opening document: %w", err)
	}

	division := models.Division{
		Index:      index,
		Designator: "",
		FullName:   "",
		Schedule:   &models.Schedule{},
	}

	designator, fullName, err := scrapeDivisionTitle(doc)
	if err != nil {
		return nil, fmt.Errorf("error scraping division title: %w", err)
	}
	division.Designator = designator
	division.FullName = fullName

	DivisionsMu.Lock()
	for _designator, _index := range DivisionsDesignators.Designators {
		if index == _index {
			fmt.Println("division's name changed, deleting old designator")
			fmt.Println("old designator:", _designator)
			delete(DivisionsDesignators.Designators, _designator)
		}
	}
	DivisionsDesignators.Designators[designator] = index
	DivisionsMu.Unlock()

	schedule, err := scrapeSchedule(doc)
	if err != nil {
		return nil, fmt.Errorf("error scraping division schedule: %w", err)
	}
	division.Schedule = schedule

	return &division, nil
}

func ScrapeTeacher(index int64) (*models.Teacher, error) {
	endpoint := makeTeacherEndpoint(index)
	doc, err := utils.OpenDoc(Config.BaseUrl, endpoint)
	if err != nil {
		return nil, fmt.Errorf("error opening document: %w", err)
	}

	teacher := models.Teacher{
		Index:      index,
		Designator: "",
		FullName:   "",
		Schedule:   &models.Schedule{},
	}

	designator, fullName, err := scrapeTeacherTitle(doc)
	if err != nil {
		return nil, fmt.Errorf("error scraping division title: %w", err)
	}
	teacher.Designator = designator
	teacher.FullName = fullName

	TeachersMu.Lock()
	for _designator, _index := range TeachersDesignators.Designators {
		if index == _index {
			fmt.Println("teacher's name changed, deleting old designator")
			delete(TeachersDesignators.Designators, _designator)
		}
	}
	TeachersDesignators.Designators[designator] = index
	TeachersMu.Unlock()

	schedule, err := scrapeSchedule(doc)
	if err != nil {
		return nil, fmt.Errorf("error scraping division schedule: %w", err)
	}
	teacher.Schedule = schedule

	return &teacher, nil
}

func ScrapeRoom(index int64) (*models.Room, error) {
	endpoint := makeRoomEndpoint(index)
	doc, err := utils.OpenDoc(Config.BaseUrl, endpoint)
	if err != nil {
		return nil, fmt.Errorf("error opening document: %w", err)
	}

	room := models.Room{
		Index:      index,
		Designator: "",
		Schedule:   &models.Schedule{},
	}

	designator, err := scrapeRoomTitle(doc)
	if err != nil {
		return nil, fmt.Errorf("error scraping division title: %w", err)
	}
	room.Designator = designator

	RoomsMu.Lock()
	for _designator, _index := range RoomsDesignators.Designators {
		if index == _index {
			fmt.Println("rooms's name changed, deleting old designator")
			delete(RoomsDesignators.Designators, _designator)
		}
	}
	RoomsDesignators.Designators[designator] = index
	RoomsMu.Unlock()

	schedule, err := scrapeSchedule(doc)
	if err != nil {
		return nil, fmt.Errorf("error scraping division schedule: %w", err)
	}
	room.Schedule = schedule

	return &room, nil
}

func ScrapeDivisionsIndexes() ([]int64, error) {
	doc, err := utils.OpenDoc(Config.BaseUrl, Config.Endpoints.DivisionsList)
	if err != nil {
		return nil, fmt.Errorf("error opening document: %w", err)
	}

	indexes := []int64{}
	doc.Find("a").Each(func(index int, element *goquery.Selection) {
		href, exists := element.Attr("href")
		if exists {
			match := DivisionIndexRegex.FindStringSubmatch(href)
			if len(match) > 1 {
				number := match[1]
				num, err := strconv.ParseInt(number, 10, 64)
				if err != nil {
					fmt.Printf("error converting number to uint: %v\n", err)
					return
				}
				endpoint := makeDivisionEndpoint(num)
				_, err = utils.OpenDoc(Config.BaseUrl, endpoint)
				if err != nil {
					fmt.Printf("error opening division document: %v\n", err)
					return
				}
				indexes = append(indexes, num)
			}
		}
	})

	return indexes, nil
}

func ScrapeTeachersIndexes() ([]int64, error) {
	doc, err := utils.OpenDoc(Config.BaseUrl, Config.Endpoints.TeachersList)
	if err != nil {
		return nil, fmt.Errorf("error opening document: %w", err)
	}

	indexes := []int64{}

	doc.Find("a").Each(func(index int, element *goquery.Selection) {
		href, exists := element.Attr("href")
		if exists {
			match := TeacherIndexRegex.FindStringSubmatch(href)
			if len(match) > 1 {
				number := match[1]
				num, err := strconv.ParseInt(number, 10, 64)
				if err != nil {
					fmt.Printf("error converting number to uint: %v\n", err)
					return
				}
				endpoint := makeTeacherEndpoint(num)
				_, err = utils.OpenDoc(Config.BaseUrl, endpoint)
				if err != nil {
					fmt.Printf("error opening teacher document: %v\n", err)
					return
				}
				indexes = append(indexes, num)
			}
		}
	})

	return indexes, nil
}

func ScrapeRoomsIndexes() ([]int64, error) {
	doc, err := utils.OpenDoc(Config.BaseUrl, Config.Endpoints.RoomsList)
	if err != nil {
		return nil, fmt.Errorf("error opening document: %w", err)
	}

	indexes := []int64{}

	doc.Find("a").Each(func(index int, element *goquery.Selection) {
		href, exists := element.Attr("href")
		if exists {
			match := RoomIndexRegex.FindStringSubmatch(href)
			if len(match) > 1 {
				number := match[1]
				num, err := strconv.ParseInt(number, 10, 64)
				if err != nil {
					fmt.Printf("error converting number to uint: %v\n", err)
					return
				}
				endpoint := makeRoomEndpoint(num)
				_, err = utils.OpenDoc(Config.BaseUrl, endpoint)
				if err != nil {
					fmt.Printf("error opening room document: %v\n", err)
					return
				}
				indexes = append(indexes, num)
			}
		}
	})

	return indexes, nil
}

func Initialize() (*models.ScheduleChannels, error) {
	fmt.Println("initializing scraper")
	Config = config.Global.Scraper

	divisionsIndexes, err := ScrapeDivisionsIndexes()
	if err != nil {
		return nil, fmt.Errorf("error scraping divisions indexes: %w", err)
	}
	DivisionsIndexes = divisionsIndexes

	teachersIndexes, err := ScrapeTeachersIndexes()
	if err != nil {
		return nil, fmt.Errorf("error scraping teachers indexes: %w", err)
	}
	TeachersIndexes = teachersIndexes

	roomsIndexes, err := ScrapeRoomsIndexes()
	if err != nil {
		return nil, fmt.Errorf("error scraping rooms indexes: %w", err)
	}
	RoomsIndexes = roomsIndexes

	divisionsIndexesLength := int64(len(DivisionsIndexes))
	teachersIndexesLength := int64(len(TeachersIndexes))
	roomsIndexesLength := int64(len(RoomsIndexes))

	fmt.Printf("starting with %d divisions, %d teachers, %d rooms\n", divisionsIndexesLength, teachersIndexesLength, roomsIndexesLength)
	
	if divisionsIndexesLength == 0 {
		fmt.Printf("no divisions found despite %d workers\n", Config.Quantities.Workers.Division)
	}
	if teachersIndexesLength == 0 {
		fmt.Printf("no teachers found despite %d workers\n", Config.Quantities.Workers.Teacher)
	}
	if roomsIndexesLength == 0 {
		fmt.Printf("no rooms found despite %d workers\n", Config.Quantities.Workers.Room)
	}

	DivisionsHub = hub.NewHub(Config.Quantities.Workers.Division)
	TeachersHub = hub.NewHub(Config.Quantities.Workers.Teacher)
	RoomsHub = hub.NewHub(Config.Quantities.Workers.Room)

	divisionsChan := ObserveDivisions()
	teachersChan := ObserveTeachers()
	roomsChan := ObserveRooms()

	return &models.ScheduleChannels{
		Divisons: divisionsChan,
		Teachers: teachersChan,
		Rooms:    roomsChan,
	}, nil
}

func Cleanup() {
	fmt.Println("cleaning scraper")
	if DivisionsHub != nil {
		DivisionsHub.Stop()
	}

	if TeachersHub != nil {
		TeachersHub.Stop()
	}

	if RoomsHub != nil {
		RoomsHub.Stop()
	}
}
