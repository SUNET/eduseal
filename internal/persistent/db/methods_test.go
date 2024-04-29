package db

import (
	"context"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"testing"

	"github.com/masv3971/gosunetca/types"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestSaveTransaction(t *testing.T) {
	t.SkipNow()
	tts := []struct {
		name  string
		tFunc func(mt *mtest.T) error //func(mt *mtest.T) *actor.Actor
		have  string
		want  string
		resp  bson.D
	}{
		{
			name: "OK",
			tFunc: func(mt *mtest.T) error {
				return nil
			},
			have: "have",
		},
	}
	opts := mtest.NewOptions().DatabaseName("wallet").ClientType(mtest.Mock)
	mt := mtest.New(t, opts)
	//defer mt.Close()
	for _, tt := range tts {
		mt.Run(tt.name, func(mt *mtest.T) {
			ctx := context.Background()

			cfg := &model.Cfg{
				Common: model.Common{},
			}

			tracer, err := trace.New(ctx, cfg, logger.NewSimple("test-tracer"), "vc", "issuer")
			assert.NoError(t, err)

			mt.AddMockResponses(tt.resp)

			//mongo := NewMongo(testMongoURI, testDbName, nil)
			s, err := New(ctx, cfg, tracer, logger.NewSimple("test-db"))
			//mongo.db = mt.DB

			// Test function
			err = s.EduSealDocumentColl.Save(context.Background(), &types.Document{})
			assert.NoError(t, err)
			//assert.Equal(t, tt.want, got)
		})
	}
}
