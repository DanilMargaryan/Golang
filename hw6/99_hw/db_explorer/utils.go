package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

func getRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	// Получаем имена всех колонок
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Массив для хранения всех результатов
	var results []map[string]interface{}

	// Создадим срез для хранения указателей на значения, которые мы будем сканировать
	values := make([]interface{}, len(columns))

	// Идём по всем строкам результата запроса
	for rows.Next() {
		// Для каждой строки создаём карту, которая будет хранить значения
		rowData := make(map[string]interface{})

		// Инициализируем указатели для каждого столбца
		for i := range values {
			var colValue interface{}
			values[i] = &colValue
		}

		// Сканируем текущую строку в значения
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}

		// Заполняем карту, используя имена колонок как ключи и значения из строки как значения
		for i, colName := range columns {
			// Значение указываем через значения из указателей в values[i]
			if val, ok := values[i].(*interface{}); ok {
				// Проверяем тип значения и преобразуем []byte в string, если это необходимо
				if byteArray, isBytes := (*val).([]byte); isBytes {
					rowData[colName] = string(byteArray)
				} else {
					rowData[colName] = *val
				}
			}
		}

		// Добавляем текущую строку в результаты
		results = append(results, rowData)
	}

	// Проверяем на ошибки при обходе строк
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

type ResultResponse struct {
	Error    string      `json:"error,omitempty"`
	Response interface{} `json:"response,omitempty"`
}

func isValidType(value interface{}, fieldType string) bool {
	switch fieldType {
	case "int", "int(11)":
		_, ok := value.(float64)
		return ok
	case "varchar", "text", "varchar(255)": // строки
		_, ok := value.(string)
		return ok
	default:
		return false
	}

}

func MarshalAndWrite(w http.ResponseWriter, status int, response interface{}) {
	result, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(status)
	w.Write(result)
}

func (explorer *DbExplore) getPrimaryKey(tableName string) (string, error) {
	fields, exists := explorer.tables[tableName]
	if !exists {
		return "", fmt.Errorf("table %s does not exist", tableName)
	}

	// Ищем поле с флагом isPrimaryKey
	for _, field := range fields {
		if field.isPrimaryKey {
			return field.name, nil
		}
	}

	return "", fmt.Errorf("primary key not found for table %s", tableName)
}
