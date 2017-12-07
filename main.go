package main

import (
	"os"
	"jsonconfig"
	"database"
	"server"
	"encoding/json"
	"fmt"
	"appinfo"
	"generator"
	"flag"
)

type configuration struct {
	Database database.Info
	Server   server.Server
	AppInfo  appinfo.AppInfo
}

func (c *configuration) ParseJSON(b []byte) error {
	return json.Unmarshal(b, &c)
}

var config = &configuration{}

func main() {

	// Get flag -s(sample data)
	includeSample := flag.Bool("s", false, "a bool")
	flag.Parse()

	// Load the configuration file
	jsonconfig.Load("config"+string(os.PathSeparator)+"config.json", config)

	// Connect to database
	database.Connect(config.Database)

	// Migrate tables
	database.SQL.AutoMigrate(&generator.Entity{},
		&generator.Column{},
		&generator.ColumnType{},
		&generator.Relation{},
		&generator.RelationType{})

	upsertRelationTypes()
	if *includeSample {
		upsertSampleData()
	}

	fmt.Println(config.AppInfo.Name, "generated!!")
}

func upsertSampleData() {
	app := &config.AppInfo

	if app == nil {
		return
	}

	for _, val := range app.FieldTypes {
		colType := generator.ColumnType{ID: val.Id, Type: val.Name}
		database.SQL.FirstOrCreate(&colType)
	}

	for i, val := range app.Entities {
		entity := generator.Entity{
			Name:        app.Entities[i].Name,
			DisplayName: app.Entities[i].DisplayName,
		}

		err := database.SQL.Create(&entity).Error
		if err == nil {

			//since gorm has no full proof way to add foreign key constraint for all db types,
			//manually checking if tables are created only then
			//add columns for those entities

			for j := range val.Fields {
				col := generator.Column{
					Name:        val.Fields[j].Name,
					DisplayName: val.Fields[j].DisplayName,
					TypeID:      val.Fields[j].Type,
					Size:        val.Fields[j].Size,
					EntityID:    entity.ID,
				}
				database.SQL.Create(&col)
			}
		}
	}

	for k, val := range app.Relations {

		parent := generator.Entity{}
		child := generator.Entity{}
		parentField := generator.Column{}
		childField := generator.Column{}

		parentErr := database.SQL.First(&parent, "name=(?)", val.ParentEntity).Error
		childErr := database.SQL.First(&child, "name=(?)", val.ChildEntity).Error

		if parentErr != nil || childErr != nil {
			return
		}

		parentFieldErr := database.SQL.First(&parentField, "name=(?) && entity_id=(?)", val.ParentEntityField, parent.ID).Error
		childFieldErr := database.SQL.First(&childField, "name=(?) && entity_id=(?)", val.ChildEntityField, child.ID).Error

		if parentFieldErr != nil || childFieldErr != nil {
			return
		}

		relation := generator.Relation{
			ParentEntityID:    parent.ID,
			ParentEntityColID: parentField.ID,
			ChildEntityID:     child.ID,
			ChildEntityColID:  child.ID,
			RelationTypeID:    app.Relations[k].Type,
		}

		database.SQL.Create(&relation)
	}

}

func upsertRelationTypes() {

	//relationship types are hardcoded because they are used in code generation, relation types in config.json is just for reference

	oneToOne := generator.RelationType{ID: 1, Name: "OneToOne"}
	oneToMany := generator.RelationType{ID: 2, Name: "OneToMany"}
	manyToMany := generator.RelationType{ID: 3, Name: "ManyToMany"}

	database.SQL.FirstOrCreate(&oneToOne)
	database.SQL.FirstOrCreate(&oneToMany)
	database.SQL.FirstOrCreate(&manyToMany)
}
