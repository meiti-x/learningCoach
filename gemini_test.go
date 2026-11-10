package main

import "testing"

func TestPostLooksComplete(t *testing.T) {
	ok := "ببین این نکته مهمه\n\nنتیجه اینه که تو پروداکشن مواظب باش\n\n🔗 [Title](https://example.com)\n\n#backend\n#linux"
	if !postLooksComplete(ok) {
		t.Fatal("expected complete")
	}
	cut := "توی اسکریپت‌نویسی"
	if postLooksComplete(cut) {
		t.Fatal("expected incomplete")
	}
	badFence := "متن\n```bash\nfind .\n\n🔗 [T](https://x)\n\n#backend\n#linux"
	if postLooksComplete(badFence) {
		t.Fatal("expected bad fence incomplete")
	}
}
