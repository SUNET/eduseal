package model

import (
	"testing"
)

func diffError(t *testing.T, name, diff string) {
	t.Errorf("Name:%s mismatch (-want +got):\n%s", name, diff)
}

//func TestMakeChannelEvents(t *testing.T) {
//	tts := []struct {
//		name string
//		have *MSG
//		want []*LadokToAggregateMSG
//	}{
//		{
//			name: "OK",
//			have: &MSG{
//				Feed: &Feed{
//					Entry: []FeedEntry{
//						{
//							ID: "testID_1",
//							Content: Content{
//								AterbudHandelse: &SDAterbudHandelse{
//									StudentUID: "testStudentUID_1",
//								},
//							},
//						},
//						{
//							ID: "testID_2",
//							Content: Content{
//								AvbrottEvent: &SDAvbrottEvent{
//									StudentUID: "testStudentUID_2",
//								},
//							},
//						},
//						{
//							ID: "testID_3",
//							Content: Content{
//								PaborjatUtbildningstillfalleEvent: &SDPaborjatUtbildningstillfalleEvent{
//									StudentUID: "testStudentUID_3",
//								},
//							},
//						},
//						{
//							ID: "testID_4",
//							Content: Content{
//								ForstagangsregistreringHandelse: &SDForstagangsregistreringHandelse{
//									StudentUID: "testStudentUID_4",
//								},
//							},
//						},
//						{
//							ID: "testID_5",
//							Content: Content{
//								StudentEvent: &SIStudentEvent{
//									StudentUID: "testStudentUID_5",
//								},
//							},
//						},
//						{
//							ID: "testID_6",
//							Content: Content{
//								StudentrestriktionEvent: &SIStudentrestriktionEvent{
//									StudentUID: "testStudentUID_6",
//								},
//							},
//						},
//						{
//							ID: "testID_7",
//							Content: Content{
//								LokalStudentEvent: &SILokalStudentEvent{
//									StudentUID: "testStudentUID_7",
//								},
//							},
//						},
//					},
//				},
//			},
//
//			want: []*LadokToAggregateMSG{
//				{
//					Payload: &EventPayload{
//						StudentUID: "testStudentUID_1",
//						EntryID:    "testID_1",
//					},
//				},
//				{
//					Payload: &EventPayload{
//						StudentUID: "testStudentUID_2",
//						EntryID:    "testID_2",
//					},
//				},
//				{
//					Payload: &EventPayload{
//						StudentUID: "testStudentUID_3",
//						EntryID:    "testID_3",
//					},
//				},
//				{
//					Payload: &EventPayload{
//						StudentUID: "testStudentUID_4",
//						EntryID:    "testID_4",
//					},
//				},
//				{
//					Payload: &EventPayload{
//						StudentUID: "testStudentUID_5",
//						EntryID:    "testID_5",
//					},
//				},
//				{
//					Payload: &EventPayload{
//						StudentUID: "testStudentUID_6",
//						EntryID:    "testID_6",
//					},
//				},
//				{
//					Payload: &EventPayload{
//						StudentUID: "testStudentUID_7",
//						EntryID:    "testID_7",
//					},
//				},
//			},
//		},
//	}
//
//	for _, tt := range tts {
//		tt.have.MakeChannelEvents()
//
//		if diff := cmp.Diff(tt.want, tt.have.Events); diff != "" {
//			diffError(t, tt.name, diff)
//		}
//	}
//}
//
//func TestAddTimestamp(t *testing.T) {
//	Now = func() time.Time {
//		return time.Date(2021, 02, 23, 20, 34, 58, 66666, time.UTC)
//	}
//
//	tts := []struct {
//		name string
//		have LadokToAggregateMSG
//		want []*EventTimestamp
//	}{
//		{
//			name: "OK",
//			have: LadokToAggregateMSG{},
//			want: []*EventTimestamp{
//				{
//					Title:     "testTitle_1",
//					Timestamp: Now(),
//				},
//				{
//					Title:     "testTitle_2",
//					Timestamp: Now(),
//				},
//			},
//		},
//	}
//
//	for _, tt := range tts {
//		tt.have.AddTimestamp("testTitle_1")
//		tt.have.AddTimestamp("testTitle_2")
//
//		if diff := cmp.Diff(tt.want, tt.have.Timestamps); diff != "" {
//			diffError(t, tt.name, diff)
//		}
//	}
//}

//func TestExtractFeed(t *testing.T) {
//	tts := []struct {
//		name string
//		have *MSG
//		want *MSG
//	}{
//		{
//			name: "OK",
//			have: &MSG{
//				Feed: &CustomTranslator{
//					Title:     "testTitleHead",
//					Generator: "http://ladok.se/uppfoljning",
//					Items: []*gofeed.Item{
//						{
//							Categories: []string{
//								fmt.Sprintf("term=%s", "se.ladok.studiedeltagande.interfaces.events.ÅterbudEvent"),
//								fmt.Sprintf("label=%s", "Event-typ"),
//							},
//							GUID:    "1",
//							Content: "application/vnd.ladok+xml",
//						},
//						{
//							Title: "testTitleEntrie2",
//							GUID:  "2",
//						},
//					},
//				},
//			},
//			want: &MSG{},
//		},
//	}
//
//	for _, tt := range tts {
//		fmt.Println(tt.have.Feed)
//	}
//}
//
