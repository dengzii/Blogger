package gen

import (
	"bytes"
	"errors"
	"github.com/CloudyKit/jet"
	"github.com/yuin/goldmark"
	"io"
	"os"
	"sort"
	"strings"
)

var templateSet *jet.Set

type RenderConfig struct {
	OutputDir   string
	TemplateDir string
}

type BlogTemplate struct {
	Name      string
	Variables map[string]interface{}
	*jet.Template
}

func (that *BlogTemplate) Init() error {
	if that.Template == nil {
		t, err := templateSet.GetTemplate(that.Name)
		if err != nil {
			return err
		}
		that.Template = t
	}
	return nil
}

func (that *BlogTemplate) Render(variables interface{}, outputPath string) error {

	if err := that.Init(); err != nil {
		return err
	}
	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if outputInfo.IsDir() {
			return errors.New("outputPath must be a file: " + outputPath)
		}
		_ = os.Remove(outputPath)
	}

	outputFileName := outputPath[strings.LastIndex(outputPath, string(os.PathSeparator))+1:]
	_ = os.MkdirAll(strings.TrimRight(outputPath, outputFileName), os.ModePerm)

	out, err := os.OpenFile(outputPath, os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	if err := that.Execute(out, nil, variables); err != nil {
		return err
	}
	return nil
}

type IndexTemplate struct {
	*BlogTemplate
}

type ArticleTemplate struct {
	*BlogTemplate
}

func (that *RenderConfig) validate() error {

	if len(that.OutputDir) == 0 {
		return errors.New("output dir not specified")
	}
	if len(that.TemplateDir) == 0 {
		return errors.New("template dir not specified")
	}

	inf, err := os.Stat(that.OutputDir)
	if err != nil {
		return err
	}
	if !inf.IsDir() {
		return errors.New("the output dir must be a directory")
	}

	inf, err = os.Stat(that.TemplateDir)
	if err != nil {
		return err
	}
	if !inf.IsDir() {
		return errors.New("the template dir must be a directory")
	}

	return nil
}

func Render(blog *Blog, config RenderConfig) error {

	if err := config.validate(); err != nil {
		// error
		return err
	}
	templateSet = jet.NewHTMLSet(config.TemplateDir)

	articleTemplate := &ArticleTemplate{&BlogTemplate{
		Name:      "template_article",
		Variables: nil,
	}}

	var category *Category
	var categoryDir string

	for i := range blog.Category {

		category = blog.Category[i]
		categoryDir = config.OutputDir + pathSep + category.AlternativeName
		if err := os.Mkdir(categoryDir, os.ModePerm); err != nil {
			if !os.IsExist(err) {
				//logger.E("gen.convert", "mkdir failed:"+categoryAlternativeName)
				return err
			}
		}

		for _, a := range category.Articles {
			if err := renderArticle(blog, articleTemplate, a, config); err != nil {
				return err
			}
		}
	}

	if err := renderIndex(blog, config); err != nil {
		return err
	}

	if err := renderFriends(blog, config); err != nil {
		return err
	}

	if err := renderAbout(blog, config); err != nil {
		return err
	}
	return nil
}

func renderArticle(blog *Blog, template *ArticleTemplate, a *Article, config RenderConfig) error {

	categoryDir := config.OutputDir + pathSep + a.Category.AlternativeName
	articleOutput := categoryDir + pathSep + a.AlternativeName + ".html"

	if strings.HasSuffix(a.file.name, ".html") {
		srcF, err := os.OpenFile(a.file.path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}
		_ = os.Remove(articleOutput)
		dstF, err := os.OpenFile(articleOutput, os.O_CREATE, os.ModePerm)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dstF, srcF); err != nil {
			return err
		}
	} else {
		cnt, err := a.file.read()
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		if err := goldmark.Convert(cnt, &buf); err != nil {
			return err
		}

		var bt []byte
		for true {
			b, e := buf.ReadByte()
			if e != nil {
				if e != io.EOF {
					return e
				} else {
					break
				}
			}
			bt = append(bt, b)
		}

		a.Content = string(bt)
		variables := struct {
			Article *Article
			Info    *BlogInfo
		}{
			Info:    blog.Info,
			Article: a,
		}

		if err := template.Render(variables, articleOutput); err != nil {
			//logger.E("gen.convert", "generate article failed :"+articleOutput)
			return err
		}
	}
	return nil
}

func renderIndex(blog *Blog, config RenderConfig) error {

	indexTemplate := IndexTemplate{&BlogTemplate{
		Name:      "template_index",
		Variables: nil,
	}}

	var allArticle []*Article

	for _, c := range blog.Category {
		for _, article := range c.Articles {
			allArticle = append(allArticle, article)
		}
	}

	sort.Slice(allArticle, func(i, j int) bool {
		ai := allArticle[i]
		aj := allArticle[j]
		return ai.file.createTime.After(aj.file.createTime)
	})

	indexOutput := config.OutputDir + pathSep + "index.html"
	var templateVariable = struct {
		Info     *BlogInfo
		Category []*Category
		Articles []*Article
	}{
		Info:     blog.Info,
		Category: blog.Category,
		Articles: allArticle,
	}
	return indexTemplate.Render(templateVariable, indexOutput)
}

func renderFriends(blog *Blog, config RenderConfig) error {

	friendsTemplate := BlogTemplate{
		Name: "template_friends",
	}
	output := config.OutputDir + pathSep + "friends.html"

	var templateVariable = struct {
		Info    *BlogInfo
		Friends []*Friend
	}{
		Info:    blog.Info,
		Friends: blog.Friends,
	}

	if err := friendsTemplate.Render(templateVariable, output); err != nil {
		return err
	}
	return nil
}

func renderAbout(blog *Blog, config RenderConfig) error {

	aboutTemplate := BlogTemplate{
		Name: "template_about",
	}
	output := config.OutputDir + pathSep + "about.html"

	var templateVariable = struct {
		Info  *BlogInfo
		About string
	}{
		Info:  blog.Info,
		About: blog.Description.Content,
	}

	if err := aboutTemplate.Render(templateVariable, output); err != nil {
		return err
	}
	return nil
}
