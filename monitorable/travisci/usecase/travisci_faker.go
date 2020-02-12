//+build faker

package usecase

import (
	"fmt"
	"time"

	"github.com/monitoror/monitoror/pkg/monitoror/faker"

	"github.com/AlekSi/pointer"

	"github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorable/travisci"
	travisCIModels "github.com/monitoror/monitoror/monitorable/travisci/models"
	"github.com/monitoror/monitoror/pkg/monitoror/utils/git"
	"github.com/monitoror/monitoror/pkg/monitoror/utils/nonempty"
)

type (
	travisCIUsecase struct {
		timeRefByProject map[string]time.Time
	}
)

var availableBuildStatus = faker.Statuses{
	{models.SuccessStatus, time.Second * 30},
	{models.FailedStatus, time.Second * 30},
	{models.CanceledStatus, time.Second * 20},
	{models.RunningStatus, time.Second * 60},
	{models.QueuedStatus, time.Second * 30},
	{models.WarningStatus, time.Second * 20},
}

func NewTravisCIUsecase() travisci.Usecase {
	return &travisCIUsecase{make(map[string]time.Time)}
}

func (tu *travisCIUsecase) Build(params *travisCIModels.BuildParams) (tile *models.Tile, err error) {
	tile = models.NewTile(travisci.TravisCIBuildTileType)
	tile.Label = fmt.Sprintf("%s\n%s", params.Repository, git.HumanizeBranch(params.Branch))

	tile.Status = nonempty.Struct(params.Status, tu.computeStatus(params)).(models.TileStatus)

	if tile.Status == models.WarningStatus {
		tile.Message = "Fake error message"
		return
	}

	tile.PreviousStatus = nonempty.Struct(params.PreviousStatus, models.SuccessStatus).(models.TileStatus)

	// Author
	if tile.Status == models.FailedStatus {
		tile.Author = &models.Author{}
		tile.Author.Name = nonempty.String(params.AuthorName, "John Doe")
		tile.Author.AvatarURL = nonempty.String(params.AuthorAvatarURL, "https://monitoror.com/assets/images/avatar.png")
	}

	if tile.Status == models.RunningStatus {
		estimatedDuration := nonempty.Duration(time.Duration(params.EstimatedDuration), time.Second*300)
		tile.Duration = pointer.ToInt64(nonempty.Int64(params.Duration, int64(tu.computeDuration(params, estimatedDuration).Seconds())))

		if tile.PreviousStatus != models.UnknownStatus {
			tile.EstimatedDuration = pointer.ToInt64(int64(estimatedDuration.Seconds()))
		} else {
			tile.EstimatedDuration = pointer.ToInt64(0)
		}
	}

	// StartedAt / FinishedAt
	if tile.Duration == nil {
		tile.StartedAt = pointer.ToTime(nonempty.Time(params.StartedAt, time.Now().Add(-time.Minute*10)))
	} else {
		tile.StartedAt = pointer.ToTime(nonempty.Time(params.StartedAt, time.Now().Add(-time.Second*time.Duration(*tile.Duration))))
	}

	if tile.Status != models.QueuedStatus && tile.Status != models.RunningStatus {
		tile.FinishedAt = pointer.ToTime(nonempty.Time(params.FinishedAt, tile.StartedAt.Add(time.Minute*5)))
	}

	return
}

func (tu *travisCIUsecase) computeStatus(params *travisCIModels.BuildParams) models.TileStatus {
	projectID := fmt.Sprintf("%s-%s-%s", params.Owner, params.Repository, params.Branch)
	value, ok := tu.timeRefByProject[projectID]
	if !ok {
		tu.timeRefByProject[projectID] = faker.GetRefTime()
	}

	return faker.ComputeStatus(value, availableBuildStatus)
}

func (tu *travisCIUsecase) computeDuration(params *travisCIModels.BuildParams, duration time.Duration) time.Duration {
	projectID := fmt.Sprintf("%s-%s-%s", params.Owner, params.Repository, params.Branch)
	value, ok := tu.timeRefByProject[projectID]
	if !ok {
		tu.timeRefByProject[projectID] = faker.GetRefTime()
	}

	return faker.ComputeDuration(value, duration)
}
