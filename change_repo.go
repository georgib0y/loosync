package main

import ()

type ChangeRepo interface {
	CreateFile(path string)
	ModifiedFile(path string)
}
