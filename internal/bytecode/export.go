package bytecode

import "lunex/internal/ast"

type ExportedChunk struct {
	Name       string
	SourceFile string
	SourceText string

	NTZOpcodes []byte
}

func EncodeExported(e *ExportedChunk) ([]byte, error) {
	chunk := &Chunk{
		Name:       e.Name,
		SourceFile: e.SourceFile,
		SourceText: e.SourceText,
	}

	return encodeNCWithNTZ(chunk, nil)
}

func EncodeExportedWithAST(e *ExportedChunk, _ *ast.Node) ([]byte, error) {
	return EncodeExported(e)
}
