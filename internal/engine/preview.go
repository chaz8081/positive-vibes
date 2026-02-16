package engine

import (
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/internal/fsutil"
	"github.com/chaz8081/positive-vibes/internal/target"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

// previewSkillInstall computes what installing a skill would do without writing
// any files. It returns a DryRunOp for the rendered SKILL.md and for each
// additional file found in sourceDir.
func previewSkillInstall(skill *schema.Skill, sourceDir, projectRoot string, t target.Target) ([]DryRunOp, error) {
	dest, err := fsutil.ResolveWithinRoot(filepath.Join(projectRoot, t.SkillDir()), skill.Name)
	if err != nil {
		return nil, err
	}

	relDest, err := filepath.Rel(projectRoot, dest)
	if err != nil {
		return nil, err
	}

	var ops []DryRunOp

	// 1. Preview SKILL.md (rendered from schema)
	rendered, err := schema.RenderSkillFile(skill)
	if err != nil {
		return nil, err
	}

	skillMdPath := filepath.Join(dest, "SKILL.md")
	existing, readErr := os.ReadFile(skillMdPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return nil, readErr
	}

	skillRelPath := filepath.Join(relDest, "SKILL.md")
	if os.IsNotExist(readErr) {
		ops = append(ops, DryRunOp{
			Action:   DryRunCreate,
			RelPath:  skillRelPath,
			Target:   t.Name(),
			Kind:     KindSkill,
			Resource: skill.Name,
		})
	} else {
		diff := unifiedDiff(skillRelPath, skillRelPath, string(existing), string(rendered))
		ops = append(ops, DryRunOp{
			Action:   DryRunUpdate,
			RelPath:  skillRelPath,
			Target:   t.Name(),
			Kind:     KindSkill,
			Resource: skill.Name,
			Diff:     diff,
		})
	}

	// 2. Walk sourceDir for additional files (skip SKILL.md)
	if sourceDir != "" {
		err = filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return err
			}
			if rel == "." || rel == "SKILL.md" {
				return nil
			}
			if d.IsDir() {
				return nil
			}

			srcData, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			fileRelPath := filepath.Join(relDest, rel)
			destPath := filepath.Join(dest, rel)

			existingData, readErr := os.ReadFile(destPath)
			if readErr != nil && !os.IsNotExist(readErr) {
				return readErr
			}

			if os.IsNotExist(readErr) {
				ops = append(ops, DryRunOp{
					Action:   DryRunCreate,
					RelPath:  fileRelPath,
					Target:   t.Name(),
					Kind:     KindSkill,
					Resource: skill.Name,
				})
			} else {
				diff := unifiedDiff(fileRelPath, fileRelPath, string(existingData), string(srcData))
				ops = append(ops, DryRunOp{
					Action:   DryRunUpdate,
					RelPath:  fileRelPath,
					Target:   t.Name(),
					Kind:     KindSkill,
					Resource: skill.Name,
					Diff:     diff,
				})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return ops, nil
}

// previewSingleFileInstall computes what installing a single-file resource
// (instruction, agent, or prompt) would do without writing any files.
func previewSingleFileInstall(name, content, sourcePath, projectRoot, resDir, suffix string, t target.Target, kind ApplyOpKind) (DryRunOp, error) {
	dest, err := fsutil.ResolveWithinRoot(filepath.Join(projectRoot, resDir), name+suffix)
	if err != nil {
		return DryRunOp{}, err
	}

	// Determine incoming data
	var incoming []byte
	if content != "" {
		incoming = []byte(content)
	} else if sourcePath != "" {
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return DryRunOp{}, err
		}
		incoming = data
	} else {
		// Nothing to preview — shouldn't happen but handle gracefully
		return DryRunOp{}, nil
	}

	relDest, err := filepath.Rel(projectRoot, dest)
	if err != nil {
		return DryRunOp{}, err
	}

	existing, readErr := os.ReadFile(dest)
	if readErr != nil && !os.IsNotExist(readErr) {
		return DryRunOp{}, readErr
	}

	if os.IsNotExist(readErr) {
		return DryRunOp{
			Action:   DryRunCreate,
			RelPath:  relDest,
			Target:   t.Name(),
			Kind:     kind,
			Resource: name,
		}, nil
	}

	diff := unifiedDiff(relDest, relDest, string(existing), string(incoming))
	return DryRunOp{
		Action:   DryRunUpdate,
		RelPath:  relDest,
		Target:   t.Name(),
		Kind:     kind,
		Resource: name,
		Diff:     diff,
	}, nil
}
