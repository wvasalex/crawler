Утилита состоит из 2х частей - краулера и парсера. 
Краулер собирает ссылки на конечные страницы с информацией, затем парсер собирает требуемые данные.

# crawler

Настройки для краулера (хост, селектор элементов меню, селектор конечных страниц и пр.) находятся в папке config в формате json. 
Краулер считывает все ссылки на категории, затем считывает пагинацию и выбирает по селектору ссылки со всех страниц.
Если указан файл с входными параметрами, краулер не будет парсить сайт, взяв данные из входного файла для формирования ссылок. (Вариант 
использования - поиск товаров по заданному списку ШК)
После завершения работы краулер записывает ссылки на интересующие страницы в файл.

# parser

Настройки для парсера (селектор элементов с данными и способ их сбора) находятся в папке config в формате json и представляют собой словарь ключ-значение. В качестве ключа используется название поля данных, значение - селектор на странице.
Парсер разбирает информацию с конечных страниц, сохраняя результаты в формате json. Ссылки для парсинга берутся из файла, который
сгенерировал краулер.

Запуск
go run *.go -app=crawler -config=CONFIG_NAME -poolsize=POOLSIZE -sleeptime=SLEEPTIME -debug=DEBUG
go run *.go -app=parser -config=CONFIG_NAME -poolsize=POOLSIZE -sleeptime=SLEEPTIME -debug=DEBUG

CONFIG_NAME - название файла без расширения в папке /config. Например, при указании -config=globus конфиг должен располагаться по
пути ./config/globus.json
POOLSIZE - количество потоков для выполнения запросов, целое число. Подавляющее большинство сайтов при POOLSIZE > 10 падает после 1k запросов, либо перестают отдавать интересующий контент. В много потоков (пробовал 100) удалось скачивать только Ашан - он не повис и не заблокировал контент.
SLEEPTIME - время ожидания перед выполнением запроса, целое число. Важно для сайтов, не выдерживающих больших нагрузок (barista, coffeebreak). Если указать 0, запросы будут отправляться без задержек.
DEBUG - включить режим отладки, булево значение. Если указан, кол-во запросов будет ограничено 10 (нужно при написании нового конфига).

## Пример с входными данными (по списку ШК)
```
{
  "crawler": { // Настройки краулера
    "root": {
      "input": "product_codes.txt", // файл с входными данными (список ШК) из папки ./input
      "origin": "https://online.globus.ru", // хост
      "start": "https://online.globus.ru/search/?q=" // префикс для конкатенации со строками из input файла
    },
    "item": {"selector": ".catalog-section__item__link"} // селектор ссылок на конечные страницы
  },
  "parser": { // Настройки парсера
    "title": {"selector": ".item-card--detail h1", "prop": "Text"},
    "category": {"selector": ".nav_breadcrumbs a", "filter": "2:0", "prop": "Text", "concatWith": "/"},
    "price": {"selector": ".item-card--detail .item-price__num meta[itemprop=\"price\"]", "filter": "Last", "prop": "Attr", "Attr": "content"},
    "description": {
      "selector": ".item-card__descr table",
      "header": "td:first-of-type",
      "value": "td+td", 
      "prop": "Table",
      "&brand": "Бренд",
      "&manufacturer": "Производитель",
      "&article": "Номер артикула"
    }
  }
}
```

## Пример без входных данных, со считыванием перелинковки
```
{
  "crawler": {
    "root": {
      "origin": "https://coffee-break.pro",
      "start": "https://coffee-break.pro"
    },
    "menu": {"selector" : ".dropdown-menu.level2 a"}, // селектор элементов меню (ссылки на первые страницы категорий)
    "pagination": {"selector": "#pagination_bottom a:not([rel])"}, // селектор ссылок на страницы (с числовыми значениями - без вперед/назад)
    "item": {"selector":".products-block .product-name"} // селектор ссылок на конечные страницы
  },
  "parser": {
    "title": {"selector": "h1.h1", "prop": "Text"},
    "category": {"selector": ".item-breadcrumb a:not(.home)", "prop": "Text", "concatWith": "/"},
    "price": {"selector": ".price [itemprop=\"price\"]", "prop": "Attr", "Attr": "content"},
    "text": {"selector": "#producttab-description", "prop": "Text"},
    "description": {
      "selector": "#producttab-datasheet",
      "header": "tr td span:not(.dotted-line)",
      "value": "tr td+td", 
      "prop": "Table",
      "&brand": "Бренд",
      "&code": "Штрих-код"
    },
    "articul": {"selector": "#product_reference [itemprop=\"sku\"]", "prop": "Attr", "Attr": "content"}
  }
}
```
