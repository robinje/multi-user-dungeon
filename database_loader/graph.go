package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "strconv"

    "github.com/dominikbraun/graph"
)

type Room struct {
    ID        int
    Area      string
    Title     string
    Narrative string
}

type Exit struct {
    From      int
    To        int
    Name      string
    Visible   bool
}

func main() {
    jsonFile, err := os.Open("rooms.json")
    if err != nil {
        fmt.Println(err)
        return
    }
    defer jsonFile.Close()

    byteValue, _ := ioutil.ReadAll(jsonFile)

    var data struct {
        Rooms map[string]struct {
            Area      string `json:"area"`
            Title     string `json:"title"`
            Narrative string `json:"narrative"`
            Exits     []struct {
                ExitName     string `json:"exit_name"`
                Visible      bool   `json:"visible"`
                TargetRoomID int    `json:"target_room_id"`
            } `json:"exits"`
        } `json:"rooms"`
    }
    json.Unmarshal(byteValue, &data)

    g := graph.New()

    for id, room := range data.Rooms {
        roomID, _ := strconv.Atoi(id)
        g.AddNode(roomID)
    }

    for id, room := range data.Rooms {
        fromID, _ := strconv.Atoi(id)
        for _, exit := range room.Exits {
            g.AddEdge(fromID, exit.TargetRoomID)
        }
    }

    fmt.Println("Graph:")
    for id, _ := range data.Rooms {
        roomID, _ := strconv.Atoi(id)
        fmt.Printf("Room %d: %s\n", roomID, data.Rooms[id].Title)
        edges, _ := g.Edges(roomID)
        for _, edge := range edges {
            fmt.Printf("  Exit to %d\n", edge.Target)
        }
    }
}
