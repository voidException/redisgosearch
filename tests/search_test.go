package tests

import (
	"github.com/sunfmin/redisgosearch"
	"github.com/sunfmin/mgodb"
	"labix.org/v2/mgo/bson"
	"testing"
	"time"
)

type Entry struct {
	Id          bson.ObjectId `bson:"_id"`
	GroupId     string
	Title       string
	Content     string
	Attachments []*Attachment
	CreatedAt   time.Time
}

type Attachment struct {
	Filename    string
	ContentType string
	CreatedAt   time.Time
}

type IndexedAttachment struct {
	Entry      *Entry
	Attachment *Attachment
}

func (this *IndexedAttachment) IndexPieces() (r []string, ais []redisgosearch.Indexable) {
	r = append(r, this.Attachment.Filename)
	return
}

func (this *IndexedAttachment) IndexEntity() (indexType string, key string, entity interface{}, rank int64) {
	key = this.Entry.Id.Hex() + this.Attachment.Filename
	indexType = "files"
	entity = this
	rank = this.Entry.CreatedAt.UnixNano()
	return
}

func (this *IndexedAttachment) IndexFilters() (r map[string]string) {
	r = make(map[string]string)
	r["group"] = this.Entry.GroupId
	return
}

func (this *Entry) MakeId() interface{} {
	if this.Id == "" {
		this.Id = bson.NewObjectId()
	}
	return this.Id
}

func (this *Entry) IndexPieces() (r []string, ais []redisgosearch.Indexable) {
	r = append(r, this.Title)
	r = append(r, this.Content)

	for _, a := range this.Attachments {
		r = append(r, a.Filename)
		ais = append(ais, &IndexedAttachment{this, a})
	}

	return
}

func (this *Entry) IndexEntity() (indexType string, key string, entity interface{}, rank int64) {
	key = this.Id.Hex()
	indexType = "entries"
	entity = this
	rank = this.CreatedAt.UnixNano()
	return
}

func (this *Entry) IndexFilters() (r map[string]string) {
	r = make(map[string]string)
	r["group"] = this.GroupId
	return
}

func TestIndexAndSearch(t *testing.T) {

	mgodb.Setup("localhost", "redisgosearch")

	client := redisgosearch.NewClient("localhost:6379", "theplant")

	e1 := &Entry{
		Id:      bson.ObjectIdHex("50344415ff3a8aa694000001"),
		GroupId: "Qortex",
		Title:   "Thread Safety",
		Content: "The connection http://google.com Send and Flush methods cannot be called concurrently with other calls to these methods. The connection Receive method cannot be called concurrently with other calls to Receive. Because the connection Do method uses Send, Flush and Receive, the Do method cannot be called concurrently with Send, Flush, Receive or Do. Unless stated otherwise, all other concurrent access is allowed.",
		Attachments: []*Attachment{
			{
				Filename:    "QORTEX UI 0.88.pdf",
				ContentType: "application/pdf",
				CreatedAt:   time.Now(),
			},
		},
		CreatedAt: time.Unix(10000, 0),
	}
	e2 := &Entry{
		Id:      bson.ObjectIdHex("50344415ff3a8aa694000002"),
		GroupId: "ASICS",
		Title:   "redis is a client for the Redis database",
		Content: "The Conn interface is the primary interface for working with Redis. Applications create connections by calling the Dial, DialWithTimeout or NewConn functions. In the future, functions will be added for creating shareded and other types of connections.",
		Attachments: []*Attachment{
			{
				Filename:    "Screen Shot 2012-08-19 at 11.52.51 AM.png",
				ContentType: "image/png",
				CreatedAt:   time.Now(),
			}, {
				Filename:    "Alternate Qortex Logo.jpg",
				ContentType: "image/jpg",
				CreatedAt:   time.Now(),
			},
		},
		CreatedAt: time.Unix(20000, 0),
	}

	mgodb.Save("entries", e1)
	client.Index(e1)

	mgodb.Save("entries", e2)
	client.Index(e2)

	var entries []*Entry
	count, err := client.Search("entries", "concurrent access", nil, 0, 10, &entries)
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Error(entries)
	}
	if entries[0].Title != "Thread Safety" {
		t.Error(entries[0])
	}

	var attachments []*IndexedAttachment
	_, err = client.Search("files", "alternate qortex", map[string]string{"group": "ASICS"}, 0, 20, &attachments)
	if err != nil {
		t.Error(err)
	}

	if attachments[0].Attachment.Filename != "Alternate Qortex Logo.jpg" || len(attachments) != 1 {
		t.Error(attachments[0])
	}

	// sort
	var sorted []*Entry
	client.Search("entries", "other", nil, 0, 10, &sorted)
	if sorted[0].Id.Hex() != "50344415ff3a8aa694000002" {
		t.Error(sorted[0])
	}
}
