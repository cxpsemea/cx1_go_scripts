package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var logger *logrus.Logger

func main() {
	logger = logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)

	logger.Info("Starting")

	httpClient := &http.Client{}

	TagName := flag.String("tag", "App", "Tag key to use - tag value will be set to the application name")
	MaxApps := flag.Int("max", 0, "The maximum number of applications that will be set in the tag, per project. Use 0 for no limit.")
	Sort := flag.Bool("sort", false, "If true: Applications will be sorted by name. If false: they will be added in the same order that is returned by the api.")
	SeparateTags := flag.Bool("separate", false, "If true: multiple tags named Key_1 ... Key_N will be set, one per application. If false: one tag will be used to store a comma-separated list of application names.")
	Clean := flag.Bool("clean", false, "If true: previous tags matching the provided tag key will be removed, otherwise they will be left untouched but may be overwritten by new values.")
	Change := flag.Bool("update", false, "No changes will be made unless this flag is set to true")
	MissingOnly := flag.Bool("missing", false, "Update only the projects that are missing the tag - useful if errors caused a few to fail, use this flag to fill the gaps")

	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)
	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	if *TagName == "" {
		logger.Fatalf("Tag parameter must be set")
	}

	if *SeparateTags {
		logger.Infof("Projects will receive new tags named %v_1 ... %v_n for each linked application", *TagName, *TagName)
	} else {
		logger.Infof("Projects will receive a new tag named %v with all linked applications in a comma-separated list", *TagName)
	}

	if *Sort {
		logger.Infof("Application names will be sorted alphabetically.")
	} else {
		logger.Infof("Application names will be added in the same order as the application IDs returned by the projects API.")
	}

	if *MaxApps > 0 {
		logger.Infof("A maximum of %d applications will be added to tags.", *MaxApps)
	} else {
		logger.Infof("All linked applications will be added to tags.")
	}

	if *Clean {
		logger.Infof("Previous tags with the key %v, or with a key matching %v_#, will be removed.", *TagName, *TagName)
	}
	if *MissingOnly {
		logger.Infof("Will only update the projects that are missing the %v tag.", *TagName)
		if *Clean {
			logger.Errorf("The 'missing' tag is incompatible with the 'clean' tag - disabling 'clean'")
			*Clean = false
		}
	} else {
		logger.Infof("Will update all projects.")
	}

	if *Change {
		logger.Warn("The 'update' flag is set - changes will be applied")
	} else {
		logger.Warn("The 'update' flag is not set - no changes will be made, but only printed to the console")
	}

	logger.Infof("Connected with %v", cx1client.String())

	projcount, err := cx1client.GetProjectCount()
	if err != nil {
		logger.Fatalf("Failed to get project count: %s", err)
	}
	projects, err := cx1client.GetProjects(projcount)
	if err != nil {
		logger.Fatalf("Failed to get projects: %s", err)
	}

	appcount, err := cx1client.GetApplicationCount()
	if err != nil {
		logger.Fatalf("Failed to get application count: %s", err)
	}
	apps, err := cx1client.GetApplications(uint(appcount))
	if err != nil {
		logger.Fatalf("Failed to get applications: %s", err)
	}

	AppsByID := make(map[string]*Cx1ClientGo.Application)
	for id, app := range apps {
		AppsByID[app.ApplicationID] = &apps[id]
	}

	keyRE := regexp.MustCompile(*TagName + "_[0-9]+")

	for _, project := range projects {
		logger.Infof("Checking project: %v", project.String())
		names := []string{}
		changed := false

		hasTagAlready := false

		for key := range project.Tags {
			if key == *TagName || keyRE.MatchString(key) {
				if *Clean {
					logger.Infof(" - Removing existing tag: %v = %v", key, project.Tags[key])
					delete(project.Tags, key)
					changed = true
				}

				hasTagAlready = true
			}
		}

		if *MissingOnly && hasTagAlready {
			logger.Infof(" - Skipping project because the 'missing' flag was set and this project already has a matching tag.")
		} else {

			for _, id := range project.Applications {
				if val, ok := AppsByID[id]; ok {
					names = append(names, val.Name)
					changed = true
				} else {
					logger.Errorf("Project %v is linked to unknown application with ID %v", project.String(), id)
				}
			}

			if len(names) == 0 {
				if len(project.Applications) > 0 {
					logger.Errorf("Project %v is linked to %d applications, but none of these were found", project.String(), len(project.Applications))
				}
			} else {
				if *Sort {
					slices.Sort(names)
				}

				max := len(names)
				if max > *MaxApps && *MaxApps != 0 {
					max = *MaxApps
					names = names[:max]
				}

				if *SeparateTags {
					for i := 0; i < max; i++ {
						tagKey := fmt.Sprintf("%v_%d", *TagName, i+1)
						project.Tags[tagKey] = names[i]
						logger.Infof(" - Adding new tag: %v = %v", tagKey, names[i])
						changed = true
					}
				} else {
					tagValue := strings.Join(names, ",")
					project.Tags[*TagName] = tagValue
					logger.Infof(" - Adding new tag: %v = %v", *TagName, tagValue)
					changed = true
				}
			}

			if changed {
				if *Change {
					if err = cx1client.UpdateProject(&project); err != nil {
						logger.Errorf("Failed to update project %v with new application tags: %s", project.String(), err)
					} else {
						logger.Infof(" - Updated project %v with new application tags: %v", project.String(), project.Tags)
					}
				} else {
					logger.Infof(" - Not applying changes, use --update to apply.")
				}
			} else {
				logger.Infof(" - Project %v required no changes.", project.String())
			}

		}
	}

	if !*Change {
		logger.Warnf("No changes were applied. To apply changes, re-run with the --update flag set.")
	}
}
