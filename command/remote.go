package command

import (
	"fmt"
	"hash/fnv"
)

func UniqueCommandName(name string, createCmd, updateCmd, deleteCmd string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(createCmd + updateCmd + deleteCmd))

	return fmt.Sprintf("%s-remote-%d", name, hash.Sum32())
}
