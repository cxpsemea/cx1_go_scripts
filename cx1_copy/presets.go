package main

import (
	"slices"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
)

func CopyPresets(cx1client1, cx1client2 *Cx1ClientGo.Cx1Client, logger *logrus.Logger) {
	srcPresets, err := cx1client1.GetAllSASTPresets()
	if err != nil {
		logger.Errorf("Failed to fetch presets from %v: %s", cx1client1.String(), err)
		return
	} else {
		for id := range srcPresets {
			if err = cx1client1.GetPresetContents(&srcPresets[id]); err != nil {
				logger.Errorf("Failed to get contents of preset %v in %v: %s", srcPresets[id].Name, cx1client1.String(), err)
			}
		}
	}

	srcQueries, err := cx1client1.GetSASTPresetQueries()
	if err != nil {
		logger.Errorf("Failed to fetch queries from %v: %s", cx1client1.String(), err)
		return
	}

	dstPresets, err := cx1client2.GetAllSASTPresets()
	if err != nil {
		logger.Errorf("Failed to fetch presets from %v: %s", cx1client2.String(), err)
		return
	} else {
		for id := range dstPresets {
			if err = cx1client2.GetPresetContents(&dstPresets[id]); err != nil {
				logger.Errorf("Failed to get contents of preset %v in %v: %s", dstPresets[id].Name, cx1client2.String(), err)
			}
		}
	}

	dstQueries, err := cx1client2.GetSASTPresetQueries()
	if err != nil {
		logger.Errorf("Failed to fetch queries from %v: %s", cx1client2.String(), err)
		return
	}

	for _, srcPreset := range srcPresets {
		if len(presetScope) > 0 && !slices.Contains(presetScope, srcPreset.Name) {
			logger.Infof("Preset %v is not in-scope", srcPreset.Name)
			continue
		} else {
			hasMatch := false
			srcCollection := srcPreset.GetSASTQueryCollection(srcQueries)
			for _, dstPreset := range dstPresets {
				if srcPreset.Name == dstPreset.Name {
					hasMatch = true
					dstCollection := dstPreset.GetSASTQueryCollection(dstQueries)

					if srcCollection.IsSubset(&dstCollection) && dstCollection.IsSubset(&srcCollection) {
						logger.Infof("Preset %v is the same between both environments", srcPreset.Name)
					} else {
						dstPreset.QueryFamilies = srcPreset.QueryFamilies
						err := cx1client2.UpdateSASTPreset(dstPreset)
						if err != nil {
							logger.Errorf("Failed to update preset %v in %v: %s", dstPreset.Name, cx1client2.String(), err)
						} else {
							logger.Infof("Preset %v updated in %v", dstPreset.Name, cx1client2.String())
						}
					}
					continue
				}
			}
			if !hasMatch {
				// logger.Infof("Preset %v is missing from %v and will be copied", srcPreset.Name, cx1client2.String())
				new_preset, err := cx1client2.CreateSASTPreset(srcPreset.Name, srcPreset.Description, srcCollection)
				if err != nil {
					logger.Errorf("Failed to create preset %v in %v: %s", srcPreset.Name, cx1client2.String(), err)
				} else {
					logger.Infof("Preset %v created in %v", new_preset.String(), cx1client2.String())
				}
			}
		}
	}

}
