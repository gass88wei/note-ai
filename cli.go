package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

func runCLI() {
	root := &cobra.Command{Use: "note-ai", Short: "AI 笔记助手"}
	root.AddCommand(&cobra.Command{Use: "import [folder]", Short: "导入文件夹", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		db, _ := NewDatabase()
		defer db.Close()
		s := NewSearchService(db)
		l := NewLLMClient(db)
		n := NewNoteService(db, s, l)
		c, err := n.ImportFolder(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("✅ 导入 %d 个文件\n", c)
		return nil
	}})
	root.AddCommand(&cobra.Command{Use: "search [query]", Short: "搜索", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		db, _ := NewDatabase()
		defer db.Close()
		s := NewSearchService(db)
		s.Init()
		rs, _ := s.Search(args[0], 5)
		for i, r := range rs {
			fmt.Printf("[%d] %s\n%s\n\n", i+1, r.Metadata["title"], r.Text)
		}
		return nil
	}})
	root.AddCommand(&cobra.Command{Use: "status", Short: "状态", Run: func(cmd *cobra.Command, args []string) {
		db, _ := NewDatabase()
		defer db.Close()
		s := NewSearchService(db)
		_, msg := s.CheckStatus()
		fmt.Println(msg)
	}})
	root.Execute()
}
