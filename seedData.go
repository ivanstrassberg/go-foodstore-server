package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func (s *PostgresStore) SeedWithData(fileName string) error {
	query := `insert into product (name, description, price, stock, rating, category_id)
	values ($1,$2,$3,$4,$5,$6)`
	readFile, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer readFile.Close()
	scanner := bufio.NewScanner(readFile)
	scanner.Split(bufio.ScanLines)
	var lines []string
	for scanner.Scan() {
		// txt := scanner.Text()
		// if numeric := regexp.MustCompile(`\d`).MatchString(txt); numeric {
		// 	conv, err := strconv.Atoi(txt)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	lines = append(lines, conv)
		// }
		lines = append(lines, scanner.Text())
	}
	for _, line := range lines {
		strs := strings.Split(line, ";")
		// fmt.Println(len(strs), strs[0])
		// fmt.Println(strs[0], strs[1], (strs[2]), strs[3], 0, strs[4])
		_, err := s.db.Exec(query, strs[0], strs[1], strs[2], strs[3], 0, strs[4])
		fmt.Println("Inserting...")
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Println("Insert Successful")
	}

	return nil
}
