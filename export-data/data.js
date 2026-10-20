const baseConfig = {
  static_url: "http://localhost:5001",
  server_url: "http://localhost:5001",
  replay_url: "http://localhost:5001",
  inplay_url: "localhost:5001",
  redisConfig: {
    hosts: ["172.21.211.215:6379"],
    username: "",
    password: "",
  },
  serverTag: "local",
};

// const baseConfig = {
//   static_url: "http://192.168.101.149:5001",
//   server_url: "http://192.168.101.149:5001",
//   replay_url: "http://192.168.101.149:5001",
//   inplay_url: "192.168.101.149:5001",
// };

const games = {
  data: [
    {
      vendorConfig: {
        gameLaunchURL: "{{server_url}}/gs2c/minilobby/start",
        gameIconsURL: "{{static_url}}/gs2c/lobby/icons",
      },
      games: [
        {
          "name": "Fortune Ace",
          "symbol": "vs1024fortune",
          "lastPlayedDate": 1744083444000,
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "财富高手"
        },
        {
          "name": "The Dragon Tiger",
          "symbol": "vs1024dtiger",
          "lastPlayedDate": 1743995670000,
          "nameZH": "龙虎榜"
        },
        {
          "name": "Mahjong Wins Bonus",
          "symbol": "vs1024mjwinbns",
          "lastPlayedDate": 1743664034000,
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "麻将大胜 Bonus"
        },
        {
          "name": "Super Joker",
          "symbol": "vs5spjoker",
          "lastPlayedDate": 1741682661000,
          "nameZH": "超炫小丑"
        },
        {
          "name": "Joker's Jewels Hot",
          "symbol": "vs10jokerhot",
          "lastPlayedDate": 1741421925000,
          "nameZH": "热火小丑珠宝"
        },
        {
          "name": "Ancient Egypt",
          "symbol": "vs10egypt",
          "lastPlayedDate": 1741229172000,
          "nameZH": "古代埃及"
        },
        {
          "name": "Joker Race",
          "symbol": "vs25jokrace",
          "lastPlayedDate": 1741227475000,
          "nameZH": "小丑赛跑"
        },
        {
          "name": "Joker King",
          "symbol": "vs25jokerking",
          "lastPlayedDate": 1741139594000,
          "nameZH": "小丑王国"
        },
        {
          "name": "Money Money Money",
          "symbol": "vs1money",
          "lastPlayedDate": 1741082988000,
          "nameZH": "钱生钱"
        },
        {
          "name": "Lucky Dragon Ball",
          "symbol": "vs1ball",
          "lastPlayedDate": 1741081431000,
          "tags": [
            "PAY_ANYWHERE"
          ],
          "nameZH": "金龙吐珠"
        },
        {
          "name": "Mahjong Wins 3 - Black Scatter",
          "symbol": "vswaysmahwblck",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "麻将大胜3 – 黑金龙耀"
        },
        {
          "name": "Gates of Olympus 1000",
          "symbol": "vs20olympx",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "奥林匹斯之门 1000"
        },
        {
          "name": "Sugar Rush 1000",
          "symbol": "vs20sugarrushx",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "极速糖果1000"
        },
        {
          "name": "Gates of Olympus",
          "symbol": "vs20olympgate",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "奥林匹斯之门"
        },
        {
          "name": "Sweet Bonanza 1000",
          "symbol": "vs20fruitswx",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "甜入心扉 1000"
        },
        {
          "name": "Starlight Princess 1000",
          "symbol": "vs20starlightx",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "星光公主 1000"
        },
        {
          "name": "5 Lions Megaways",
          "symbol": "vswayslions",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "5金狮 Megaways"
        },
        {
          "name": "Sweet Bonanza",
          "symbol": "vs20fruitsw",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "甜入心扉"
        },
        {
          "name": "Starlight Princess",
          "symbol": "vs20starlight",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "星光公主"
        },
        {
          "name": "Gates of Olympus Xmas 1000",
          "symbol": "vs20olympxmas",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "奥林匹斯之门圣诞 1000"
        },
        {
          "name": "Mahjong Wins 2",
          "symbol": "vswaysmahwin2",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "麻将大胜 2"
        },
        {
          "name": "Sugar Rush",
          "symbol": "vs20sugarrush",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "极速糖果"
        },
        {
          "name": "Panda's Fortune",
          "symbol": "vs25pandagold",
          "nameZH": "熊猫财富"
        },
        {
          "name": "Power of Merlin Megaways",
          "symbol": "vswayspowzeus",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "法力梅林Megaways"
        },
        {
          "name": "Gates of Gatot Kaca 1000",
          "symbol": "vs20gatotx",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "印尼传奇迦多铎卡 1000"
        },
        {
          "name": "Zeus vs Hades - Gods of War",
          "symbol": "vs15godsofwar",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "宙斯vs哈迪斯-众神之战"
        },
        {
          "name": "5 Rabbits Megaways",
          "symbol": "vswaysrabbits",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "5金兔 Megaways"
        },
        {
          "name": "Vampy Party",
          "symbol": "vswayswbounty",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "吸血鬼派对"
        },
        {
          "name": "Peppe's Pepperoni Pizza Plaza",
          "symbol": "vswaystonypzz",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Ancient Island Megaways",
          "symbol": "vswaysmodfr",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Big Bass Return to the Races",
          "symbol": "vs10bbrttr",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Savannah Legend",
          "symbol": "vswayssavlgnd",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Bigger Bass Splash",
          "symbol": "vs12bgrbspl",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "John Hunter and Galileo's Secrets",
          "symbol": "vs10booklight",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Escape the Pyramid - Fire & Ice",
          "symbol": "vs10fireice",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Wild Wildebeest Wins",
          "symbol": "vswaysbufstamp",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Wild Wild Pearls",
          "symbol": "vswayspearls",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Irish Crown",
          "symbol": "vs20irishcrown",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Aztec Gems Megaways",
          "symbol": "vswaysaztec",
          "tags": [
            "MEGAWAYS"
          ],
          "hidePPLogo": true
        },
        {
          "name": "Brick House Bonanza",
          "symbol": "vswaysbrickhos",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Floating Dragon - Year of the Snake",
          "symbol": "vs10fdsnake",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Fonzo's Feline Fortunes",
          "symbol": "vs10fonzofff",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Aztec Smash",
          "symbol": "vs20plsmcannon",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Big Bass Bonanza 3 Reeler",
          "symbol": "vs5bb3reeler",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Mining Rush",
          "symbol": "vs20minerush",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Money Stacks Megaways",
          "symbol": "vswaysbblitz",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "hidePPLogo": true
        },
        {
          "name": "Dragon King Hot Pots",
          "symbol": "vs10dkinghp",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Santa's Xmas Rush",
          "symbol": "vs20rainbowrsh",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "极速圣诞快乐"
        },
        {
          "name": "Tiny Toads",
          "symbol": "vs50fatfrogs",
          "tags": [
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Penguins Christmas Party Time",
          "symbol": "vs25xmasparty",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "企鹅圣诞派对"
        },
        {
          "name": "Might of Freya Megaways",
          "symbol": "vswaysmfreya",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "女神弗蕾雅Megaways"
        },
        {
          "name": "Joker's Jewels",
          "symbol": "vs5joker",
          "nameZH": "小丑珠宝"
        },
        {
          "name": "Sugar Rush Xmas",
          "symbol": "vs20sugrux",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "极速糖果圣诞"
        },
        {
          "name": "Aztec Gems",
          "symbol": "vs5aztecgems",
          "nameZH": "古时代宝石"
        },
        {
          "name": "The Dog House Megaways",
          "symbol": "vswaysdogs",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "汪汪之家Megaways"
        },
        {
          "name": "The Dog House",
          "symbol": "vs20doghouse",
          "nameZH": "汪汪之家"
        },
        {
          "name": "Wisdom of Athena 1000",
          "symbol": "vs20procountx",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "雅典娜的智慧 1000"
        },
        {
          "name": "Sweet Bonanza Xmas",
          "symbol": "vs20sbxmas",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "甜心盛宴圣诞"
        },
        {
          "name": "Wild West Gold",
          "symbol": "vs40wildwest",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "西部牛仔黄金地段"
        },
        {
          "name": "The Tweety House",
          "symbol": "vs20tweethouse",
          "nameZH": "禽鸟之屋"
        },
        {
          "name": "Pyramid Bonanza",
          "symbol": "vs20pbonanza",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂金字塔"
        },
        {
          "name": "Starlight Christmas",
          "symbol": "vs20schristmas",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "星光圣诞系列"
        },
        {
          "name": "Buffalo King Megaways",
          "symbol": "vswaysbufking",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "王者野牛 Megaways"
        },
        {
          "name": "Fortune of Giza",
          "symbol": "vs20amuleteg",
          "nameZH": "埃及财富"
        },
        {
          "name": "Pompeii Megareels Megaways",
          "symbol": "vswaysmegareel",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "庞贝古城 Megaways"
        },
        {
          "name": "Club Tropicana",
          "symbol": "vs12tropicana",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "热带俱乐部"
        },
        {
          "name": "Great Rhino Megaways",
          "symbol": "vswaysrhino",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "巨大犀牛 Megaways"
        },
        {
          "name": "Saiyan Mania",
          "symbol": "vs20saiman",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "hidePPLogo": true,
          "nameZH": "赛亚狂热"
        },
        {
          "name": "Anime Mecha Megaways",
          "symbol": "vswaysanime",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "机甲战队 Megaways"
        },
        {
          "name": "Panda Fortune 2",
          "symbol": "vs25pandatemple",
          "nameZH": "熊猫财富2"
        },
        {
          "name": "Rujak Bonanza",
          "symbol": "vs20rujakbnz",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "水果沙拉盛宴"
        },
        {
          "name": "Chests of Cai Shen",
          "symbol": "vs25checaishen",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "财神聚宝盒"
        },
        {
          "name": "Rocket Blast Megaways",
          "symbol": "vswaysrockblst",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "火箭发射 Megaways"
        },
        {
          "name": "Power of Thor Megaways",
          "symbol": "vswayshammthor",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "雷神之锤 Megaways"
        },
        {
          "name": "Big Bass Splash",
          "symbol": "vs10txbigbass",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "大鲈鱼溅水"
        },
        {
          "name": "PIZZA PIZZA PIZZA",
          "symbol": "vswayspizza",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "比萨！比萨？比萨！"
        },
        {
          "name": "Rock Vegas",
          "symbol": "vs20rockvegas",
          "nameZH": "石头族赌城"
        },
        {
          "name": "Fire Portals",
          "symbol": "vs20portals",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "火焰之界"
        },
        {
          "name": "Bigger Bass Bonanza",
          "symbol": "vs12bbb",
          "nameZH": "超疯狂大鲈鱼"
        },
        {
          "name": "Zombie Train",
          "symbol": "vs15seoultrain",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "首尔列车"
        },
        {
          "name": "Madame Destiny Megaways",
          "symbol": "vswaysmadame",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "命运女巫Megaways"
        },
        {
          "name": "Muertos Multiplier Megaways",
          "symbol": "vs20muertos",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "墨西哥亡灵节 Megaways"
        },
        {
          "name": "Rise of Samurai 4",
          "symbol": "vs15samurai4",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "崛起的武士4"
        },
        {
          "name": "Big Bass - Hold & Spinner",
          "symbol": "vs10bbhas",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼持控旋转"
        },
        {
          "name": "Wisdom of Athena",
          "symbol": "vs20procount",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "雅典娜的智慧"
        },
        {
          "name": "Gold Party",
          "symbol": "vs25goldparty",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "黄金派对"
        },
        {
          "name": "Sweet Bonanza Dice",
          "symbol": "vs20bnnzdice",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "甜入心扉骰子"
        },
        {
          "name": "Floating Dragon",
          "symbol": "vs10floatdrg",
          "nameZH": "鱼跃龙门 持控旋转"
        },
        {
          "name": "Christmas Carol Megaways",
          "symbol": "vs20xmascarol",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "圣诞颂歌 Megaways"
        },
        {
          "name": "Wild Wild Riches Megaways",
          "symbol": "vswayswwriches",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "爱尔兰人宝藏 Megaways"
        },
        {
          "name": "Cleocatra",
          "symbol": "vs20cleocatra",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "埃及神猫"
        },
        {
          "name": "Gates of Gatot Kaca",
          "symbol": "vs20gatotgates",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "印尼传奇加多铎卡"
        },
        {
          "name": "Clover Gold",
          "symbol": "vs20mustanggld2",
          "nameZH": "黄金三叶草"
        },
        {
          "name": "Aztec King Megaways",
          "symbol": "vswaysaztecking",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "阿兹特克国王Megaways"
        },
        {
          "name": "Barn Festival",
          "symbol": "vs20farmfest",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "农场嘉年华"
        },
        {
          "name": "Mammoth Gold Megaways",
          "symbol": "vs20mammoth",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "猛犸黄金 Megaways"
        },
        {
          "name": "Buffalo King",
          "symbol": "vs4096bufking",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "王者野牛"
        },
        {
          "name": "Mahjong Wins",
          "symbol": "vs1024mahjwins",
          "tags": [
            "TUMBLING"
          ],
          "nameZH": "麻将大胜"
        },
        {
          "name": "Hot Fiesta",
          "symbol": "vs25hotfiesta",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "狂热嘉年华"
        },
        {
          "name": "Fruits of the Amazon",
          "symbol": "vs20framazon",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "亚马逊果实"
        },
        {
          "name": "Big Bass Bonanza",
          "symbol": "vs10bbbonanza",
          "nameZH": "疯狂大鲈鱼"
        },
        {
          "name": "Lucky Fishing Megaways",
          "symbol": "vswaysluckyfish",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "幸运捕鱼 Megaways"
        },
        {
          "name": "Fruit Party",
          "symbol": "vs20fruitparty",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ]
        },
        {
          "name": "Big Bass Halloween",
          "symbol": "vs10bhallbnza",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼万圣节"
        },
        {
          "name": "Big Bass Vegas Double Down Deluxe",
          "symbol": "vs10bbdoubled",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "至尊双赢疯狂大鲈鱼"
        },
        {
          "name": "Gates of Olympus Dice",
          "symbol": "vs20olympdice",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "奥林匹斯之门骰子"
        },
        {
          "name": "Fish Eye",
          "symbol": "vs10fisheye",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "埃及鱼之眼"
        },
        {
          "name": "Bigger Bass Blizzard - Christmas Catch",
          "symbol": "vs12bbbxmas",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "超级大鲈鱼暴风雨 - 圣诞鱼获"
        },
        {
          "name": "Return of the Dead",
          "symbol": "vs10returndead",
          "nameZH": "活死人归来"
        },
        {
          "name": "Rise of Samurai Megaways",
          "symbol": "vswayssamurai",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "崛起的武士 Megaways"
        },
        {
          "name": "Floating Dragon - Dragon Boat Festival",
          "symbol": "vs10fdrasbf",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "鱼跃龙门 天空之船"
        },
        {
          "name": "Fire Stampede",
          "symbol": "vswaysstampede",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "热火暴走"
        },
        {
          "name": "Cash Patrol",
          "symbol": "vs25copsrobbers",
          "nameZH": "巡逻现警"
        },
        {
          "name": "Big Bass Floats my Boat",
          "symbol": "vs10bbfloats",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼摇摇船"
        },
        {
          "name": "Power of Ninja",
          "symbol": "vs20ninjapower",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "武力忍者"
        },
        {
          "name": "Christmas Big Bass Bonanza",
          "symbol": "vs10bxmasbnza",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "圣诞大鲈鱼"
        },
        {
          "name": "Holiday Ride",
          "symbol": "vs25holiday",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "兜风假期"
        },
        {
          "name": "Wild West Gold Megaways",
          "symbol": "vswayswildwest",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "西部牛仔黄金地段 Megaways"
        },
        {
          "name": "Sugar Supreme Powernudge",
          "symbol": "vs20sugarnudge",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "顶级糖果Powernudge"
        },
        {
          "name": "The Dog House - Muttley Crew",
          "symbol": "vs20dhcluster2",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "汪汪之家 – 穆特利伙伴"
        },
        {
          "name": "Samurai Code",
          "symbol": "vs12scode",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "武士密码"
        },
        {
          "name": "Wild Celebrity Bus Megaways",
          "symbol": "vswaysrsm",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "狂野名人大巴 Megaways"
        },
        {
          "name": "The Dog House - Dog or Alive",
          "symbol": "vs20doghouse2",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "汪汪之家 狗狗神气"
        },
        {
          "name": "5 Lions Gold",
          "symbol": "vs243lionsgold",
          "nameZH": "5金狮"
        },
        {
          "name": "Release the Kraken Megaways",
          "symbol": "vswayskrakenmw",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "解放大海怪 Megaways"
        },
        {
          "name": "Angel vs Sinner",
          "symbol": "vs15fghtmultlv",
          "nameZH": "天使与罪人"
        },
        {
          "name": "Mochimon",
          "symbol": "vs20mochimon",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "麻糬宝贝"
        },
        {
          "name": "The Dog House Multihold",
          "symbol": "vs20doghousemh",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "汪汪之家多方位持控"
        },
        {
          "name": "3 Buzzing Wilds",
          "symbol": "vs20wildparty",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "喧闹虫虫"
        },
        {
          "name": "Mustang Gold Megaways",
          "symbol": "vswaysgoldcol",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "黄金野马 Megaways"
        },
        {
          "name": "Big Bass - Secrets of the Golden Lake",
          "symbol": "vs10bblotgl",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼 黄金湖的秘密"
        },
        {
          "name": "Candy Blitz Bombs",
          "symbol": "vs20candybltz2",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "闪电糖果轰炸"
        },
        {
          "name": "Spin & Score Megaways",
          "symbol": "vswaysfrywld",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "旋转及射龙门Megaways"
        },
        {
          "name": "Rabbit Garden",
          "symbol": "vs20goldclust",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "hidePPLogo": true,
          "nameZH": "兔子乐园"
        },
        {
          "name": "Forge of Olympus",
          "symbol": "vs20forge",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "奥林匹斯锻造"
        },
        {
          "name": "The Wild Gang",
          "symbol": "vswayswildgang",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "野蛮帮派"
        },
        {
          "name": "Casino Heist Megaways",
          "symbol": "vswayscheist",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "娱乐场大劫案 Megaways"
        },
        {
          "name": "7 Clovers of Fortune",
          "symbol": "vswayssevenc",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "7幸运草"
        },
        {
          "name": "Lucky Lightning",
          "symbol": "vswayslight",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "幸运雷神"
        },
        {
          "name": "Wild West Duels",
          "symbol": "vs20pistols",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "西部牛仔对决"
        },
        {
          "name": "Pirate Gold",
          "symbol": "vs40pirate",
          "nameZH": "夺金海贼"
        },
        {
          "name": "Ancient Egypt Classic",
          "symbol": "vs10egyptcls",
          "nameZH": "古代埃及经典版"
        },
        {
          "name": "Extra Juicy",
          "symbol": "vs10fruity2",
          "nameZH": "额外多汁"
        },
        {
          "name": "Kingdom of Asgard",
          "symbol": "vs20asgard",
          "nameZH": "阿斯加德之王"
        },
        {
          "name": "Wild Bison Charge",
          "symbol": "vs20stickywild",
          "tags": [
            "PAY_ANYWHERE",
            "BUY_FEATURE"
          ],
          "nameZH": "野牛冲锋"
        },
        {
          "name": "Big Bass Bonanza - Keeping it Reel",
          "symbol": "vs10bbkir",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼持续旋转"
        },
        {
          "name": "Eye of the Storm",
          "symbol": "vs10eyestorm",
          "nameZH": "风暴之眼"
        },
        {
          "name": "Viking Forge",
          "symbol": "vs20sugarcoins",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "维京锻造"
        },
        {
          "name": "Happy Fortune",
          "symbol": "vs20laughluck",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "笑口迎财"
        },
        {
          "name": "Gems Bonanza",
          "symbol": "vs20goldfever",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "富矿宝石"
        },
        {
          "name": "Sticky Bees",
          "symbol": "vs20clustwild",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "粘性蜜蜂"
        },
        {
          "name": "Wukong Rush",
          "symbol": "vs20mkrush",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "极速美猴王"
        },
        {
          "name": "Great Rhino Deluxe",
          "symbol": "vs20rhinoluxe",
          "nameZH": "巨大犀牛 加强版"
        },
        {
          "name": "Zombie Carnival",
          "symbol": "vswayszombcarn",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "丧尸嘉年华"
        },
        {
          "name": "Tic Tac Take",
          "symbol": "vs10tictac",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "井字游戏"
        },
        {
          "name": "American Blackjack",
          "symbol": "bjmb"
        },
        {
          "name": "5 Frozen Charms Megaways",
          "symbol": "vswayscharms",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "5冰冻魅力Megaways"
        },
        {
          "name": "Fire Strike",
          "symbol": "vs10firestrike",
          "nameZH": "红火暴击"
        },
        {
          "name": "Book of Fallen",
          "symbol": "vs10bookfallen",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "坠落之书"
        },
        {
          "name": "Grace of Ebisu",
          "symbol": "vs20olympgrace",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "七福海神"
        },
        {
          "name": "Fruity Treats",
          "symbol": "vs20fortbon",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "hidePPLogo": true,
          "nameZH": "水果盛宴"
        },
        {
          "name": "Hand of Midas 2",
          "symbol": "vs20midas2",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "迈达斯之手 2"
        },
        {
          "name": "Extra Juicy Megaways",
          "symbol": "vswaysxjuicy",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "额外多汁Megaways"
        },
        {
          "name": "Aztec Gems Deluxe",
          "symbol": "vs9aztecgemsdx",
          "nameZH": "古时代宝石 加强版"
        },
        {
          "name": "Buffalo King Untamed Megaways",
          "symbol": "vswaysbkingasc",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "王者野牛奔放Megaways"
        },
        {
          "name": "Ultra Burn",
          "symbol": "vs5ultrab",
          "nameZH": "超级燃烧"
        },
        {
          "name": "6 Jokers",
          "symbol": "vs5magicdoor",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING"
          ],
          "nameZH": "6小丑"
        },
        {
          "name": "3 Dancing Monkeys",
          "symbol": "vswaysmonkey",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "3舞蹈猴子"
        },
        {
          "name": "Santa's Great Gifts",
          "symbol": "vs20porbs",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "圣诞惊喜礼物"
        },
        {
          "name": "The Dog House Dice Show",
          "symbol": "vs20dhdice",
          "nameZH": "汪汪之家骰子秀"
        },
        {
          "name": "Excalibur Unleashed",
          "symbol": "vs20excalibur",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "圣剑出鞘"
        },
        {
          "name": "Big Bass Christmas Bash",
          "symbol": "vs10bbsplxmas",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼圣诞节"
        },
        {
          "name": "Legend of Heroes Megaways",
          "symbol": "vswayslofhero",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "英雄联盟 Megaways"
        },
        {
          "name": "Big Bass Halloween 2",
          "symbol": "vs10bhallbnza2",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼万圣节2"
        },
        {
          "name": "Candy Blitz",
          "symbol": "vs20candyblitz",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "糖果闪电"
        },
        {
          "name": "Fury of Odin Megaways",
          "symbol": "vswaysfuryodin",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "奥丁之怒 Megaways"
        },
        {
          "name": "Cosmic Cash",
          "symbol": "vs40cosmiccash",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "宇宙现金"
        },
        {
          "name": "Big Bass Xmas Xtreme",
          "symbol": "vs10bbxext",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼圣诞极限"
        },
        {
          "name": "Big Bass Bonanza - Reel Action",
          "symbol": "vs10bbbrlact",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼 – 即刻出动"
        },
        {
          "name": "Cash Elevator",
          "symbol": "vs20terrorv",
          "nameZH": "现金升降机"
        },
        {
          "name": "Hot Pepper",
          "symbol": "vs20dugems",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "辣椒狂热"
        },
        {
          "name": "Great Rhino",
          "symbol": "vs20rhino",
          "nameZH": "巨大犀牛"
        },
        {
          "name": "Monkey Warrior",
          "symbol": "vs243mwarrior",
          "nameZH": "酷猴战士"
        },
        {
          "name": "888 Dragons",
          "symbol": "vs1dragon8",
          "nameZH": "发发发龙"
        },
        {
          "name": "Money Roll",
          "symbol": "cs5moneyroll",
          "nameZH": "财源滚滚"
        },
        {
          "name": "Hot to Burn Hold and Spin",
          "symbol": "vs20hburnhs",
          "hidePPLogo": true,
          "nameZH": "热火燃烧持控旋转"
        },
        {
          "name": "Big Bass Amazon Xtreme",
          "symbol": "vs10bbextreme",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼 亚马逊极限"
        },
        {
          "name": "Heart of Cleopatra",
          "symbol": "vs20heartcleo",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "埃及艳后之心"
        },
        {
          "name": "Cyber Heist",
          "symbol": "vs20cbrhst",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "网络大劫案"
        },
        {
          "name": "The Hand of Midas",
          "symbol": "vs20midas",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "迈达斯之手"
        },
        {
          "name": "Bow of Artemis",
          "symbol": "vs20gembondx",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "狩猎女神之弓"
        },
        {
          "name": "Big Bass Mission Fishin'",
          "symbol": "vs10bbfmission",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼垂钓任务"
        },
        {
          "name": "Magician's Secrets",
          "symbol": "vs4096magician",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "神秘的巫师"
        },
        {
          "name": "Aztec Bonanza",
          "symbol": "vs7776aztec",
          "tags": [
            "TUMBLING"
          ],
          "nameZH": "阿兹特克获利之道"
        },
        {
          "name": "Twilight Princess",
          "symbol": "vs20dhcluster",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "暮光公主"
        },
        {
          "name": "Gates of Aztec",
          "symbol": "vs20aztecgates",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "阿茲特克之门"
        },
        {
          "name": "Fruit Party 2",
          "symbol": "vs20fparty2",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "水果派对 2"
        },
        {
          "name": "John Hunter and the Tomb of the Scarab Queen",
          "symbol": "vs25scarabqueen",
          "nameZH": "金龟子女王"
        },
        {
          "name": "Ultra Hold and Spin",
          "symbol": "vs5ultra",
          "nameZH": "超级持控旋转"
        },
        {
          "name": "Phoenix Forge",
          "symbol": "vs20phoenixf",
          "tags": [
            "TUMBLING"
          ],
          "nameZH": "霹雳凤凰"
        },
        {
          "name": "Yum Yum Powerways",
          "symbol": "vswaysyumyum",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "可口好吃 Powerways"
        },
        {
          "name": "Crystal Caverns Megaways",
          "symbol": "vswayscryscav",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "水晶洞穴Megaways"
        },
        {
          "name": "Santa's Wonderland",
          "symbol": "vs20santawonder",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "圣诞仙境"
        },
        {
          "name": "Peak Power",
          "symbol": "vs10powerlines",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "极限能量"
        },
        {
          "name": "Multihand Blackjack",
          "symbol": "bjma"
        },
        {
          "name": "8 Dragons",
          "symbol": "vs20eightdragons",
          "nameZH": "8条龙"
        },
        {
          "name": "Floating Dragon Hold & Spin Megaways",
          "symbol": "vswaysfltdrg",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "鱼跃龙门 Megaways"
        },
        {
          "name": "5 Lions",
          "symbol": "vs243lions",
          "nameZH": "5雄狮"
        },
        {
          "name": "Dragon Gold 88",
          "symbol": "vs10dgold88",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "金龙88"
        },
        {
          "name": "Dragon Hero",
          "symbol": "vs20drgbless",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "神龙大侠"
        },
        {
          "name": "The Big Dawgs",
          "symbol": "vs20bigdawgs",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "大人物"
        },
        {
          "name": "Sumo Supreme Megaways",
          "symbol": "vswaysmegwghts",
          "tags": [
            "TUMBLING",
            "MEGAWAYS"
          ],
          "nameZH": "相扑力士 Megaways"
        },
        {
          "name": "Cowboys Gold",
          "symbol": "vs10cowgold",
          "nameZH": "黄金叛变"
        },
        {
          "name": "Fortunes of Aztec",
          "symbol": "vswaysstrlght",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "阿兹特克财宝"
        },
        {
          "name": "Candy Jar Clusters",
          "symbol": "vs20cjcluster",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "糖果罐派对"
        },
        {
          "name": "Congo Cash",
          "symbol": "vs432congocash",
          "nameZH": "刚果财富"
        },
        {
          "name": "Himalayan Wild",
          "symbol": "vs5himalaw",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "喜马拉雅疯狂"
        },
        {
          "name": "Release the Bison",
          "symbol": "vs20bison",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": " 释放野牛"
        },
        {
          "name": "Wild Wild Riches",
          "symbol": "vs576treasures",
          "nameZH": "爱尔兰人宝藏"
        },
        {
          "name": "Infective Wild",
          "symbol": "vs40infwild",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "狂野感染"
        },
        {
          "name": "Monster Superlanche",
          "symbol": "vs20superlanche",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "Monster Superlanche"
        },
        {
          "name": "Mahjong X",
          "symbol": "vs20mahjxbnz",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "麻将X "
        },
        {
          "name": "Fire Strike 2",
          "symbol": "vs10firestrike2",
          "nameZH": "红火暴击2"
        },
        {
          "name": "Treasure Wild",
          "symbol": "vs20trsbox",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂宝藏"
        },
        {
          "name": "Robber Strike",
          "symbol": "vs4096robber",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "大盗反击"
        },
        {
          "name": "Cowboy Coins",
          "symbol": "vswaysultrcoin",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "牛仔金币"
        },
        {
          "name": "Octobeer Fortunes",
          "symbol": "vs20octobeer",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "十月啤酒嘉年华"
        },
        {
          "name": "Super 7s ",
          "symbol": "vs5super7",
          "nameZH": "超级7s"
        },
        {
          "name": "Spirit of Adventure",
          "symbol": "vs10spiritadv",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "寻宝冒险精神"
        },
        {
          "name": "Eternal Empress - Freeze Time",
          "symbol": "vswaysfreezet",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "永恒女皇 - 急冻时刻"
        },
        {
          "name": "Master Joker",
          "symbol": "vs1masterjoker",
          "tags": [
            "PAY_ANYWHERE"
          ],
          "nameZH": "杰克大师"
        },
        {
          "name": "Reel Banks",
          "symbol": "vs25rlbank",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "摇滚银行"
        },
        {
          "name": "Queenie",
          "symbol": "vs243queenie",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "女王"
        },
        {
          "name": "Mustang Gold",
          "symbol": "vs25mustang",
          "nameZH": "黄金野马"
        },
        {
          "name": "Revenge of Loki Megaways",
          "symbol": "vswaysloki",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "洛基复仇Megaways"
        },
        {
          "name": "Spartan King",
          "symbol": "vs40spartaking",
          "nameZH": "斯巴达之王"
        },
        {
          "name": "Firebird Spirit - Connect & Collect",
          "symbol": "vswaysconcoll",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "火鸟精神"
        },
        {
          "name": "Wild Spells",
          "symbol": "vs25wildspells",
          "nameZH": "法力无边"
        },
        {
          "name": "Roulette",
          "symbol": "rla"
        },
        {
          "name": "Chilli Heat",
          "symbol": "vs25chilli",
          "nameZH": "火辣辣"
        },
        {
          "name": "Big Bass Hold & Spinner Megaways",
          "symbol": "vswaysbbhas",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼持控旋转Megaways"
        },
        {
          "name": "Candy Corner",
          "symbol": "vs20fourmc",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "糖果堆"
        },
        {
          "name": "Emperor Caishen",
          "symbol": "vs243empcaishen",
          "nameZH": "财神驾到"
        },
        {
          "name": "Aztec King",
          "symbol": "vs25aztecking",
          "nameZH": "阿茲特克国王"
        },
        {
          "name": "Yeti Quest",
          "symbol": "vs20mesmult",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "雪人探索"
        },
        {
          "name": "Black Bull",
          "symbol": "vs20trswild2",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "黑公牛"
        },
        {
          "name": "Gods of Giza",
          "symbol": "vs10gizagods",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "吉萨之神"
        },
        {
          "name": "Big Bass Day at the Races",
          "symbol": "vs10bbbnz",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂大鲈鱼竞赛"
        },
        {
          "name": "Blazing Wilds Megaways",
          "symbol": "vswaysfirewmw",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "热火燎原 Megaways"
        },
        {
          "name": "Elemental Gems Megaways",
          "symbol": "vswayselements",
          "tags": [
            "MEGAWAYS"
          ],
          "nameZH": "五行宝石 Megaways"
        },
        {
          "name": "Bounty Gold",
          "symbol": "vs25btygold",
          "nameZH": "赏金猎人"
        },
        {
          "name": "Fangtastic Freespins",
          "symbol": "vs10fangfree",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "超劲爆免费旋转"
        },
        {
          "name": "Mustang Trail",
          "symbol": "vs10trail",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "野马踪迹"
        },
        {
          "name": "Gorilla Mayhem",
          "symbol": "vs1024gmayhem",
          "tags": [
            "BUY_FEATURE"
          ],
          "hidePPLogo": true,
          "nameZH": "暴怒猩猩"
        },
        {
          "name": "Tree of Riches",
          "symbol": "vs1fortunetree",
          "nameZH": "发发树"
        },
        {
          "name": "Wolf Gold Ultimate",
          "symbol": "vs25ultwolgol",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "野狼黄金终极"
        },
        {
          "name": "Dragon Hot Hold & Spin",
          "symbol": "vs5drhs",
          "nameZH": "舞龙 热控旋转"
        },
        {
          "name": "Floating Dragon New Year Festival Ultra Megaways Hold & Spin",
          "symbol": "vswaysfltdrgny",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "鱼跃龙门贺新年Megaways"
        },
        {
          "name": "888 Bonanza",
          "symbol": "vs243goldfor",
          "nameZH": "888富矿"
        },
        {
          "name": "Release the Kraken",
          "symbol": "vs20kraken",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "解放大海怪"
        },
        {
          "name": "Diamond Strike",
          "symbol": "vs15diamond",
          "nameZH": "钻石罢工"
        },
        {
          "name": "Big Bass Bonanza Megaways",
          "symbol": "vswaysbbb",
          "tags": [
            "TUMBLING",
            "MEGAWAYS"
          ],
          "nameZH": "疯狂大鲈鱼 Megaways"
        },
        {
          "name": "Barnyard Megahays Megaways",
          "symbol": "vswaysmegahays",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂谷仓 Megaways"
        },
        {
          "name": "Sweet Kingdom",
          "symbol": "vs20clustcol",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "甜心王国"
        },
        {
          "name": "Heart of Rio",
          "symbol": "vs25rio",
          "nameZH": "裏約之心"
        },
        {
          "name": "Raging Bull",
          "symbol": "vs243chargebull",
          "nameZH": "牛转钱坤"
        },
        {
          "name": "Caishen's Gold",
          "symbol": "vs243fortune",
          "nameZH": "财神黄金"
        },
        {
          "name": "Caishen's Cash",
          "symbol": "vs243caishien",
          "nameZH": "财神运财"
        },
        {
          "name": "Tropical Tiki",
          "symbol": "vswaysjkrdrop",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "热带提基"
        },
        {
          "name": "The Money Men Megaways",
          "symbol": "vswaysmoneyman",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "大富豪Megaways"
        },
        {
          "name": "Monkey Madness",
          "symbol": "vs9madmonkey",
          "nameZH": "猴子疯狂"
        },
        {
          "name": "Dance Party",
          "symbol": "vs243dancingpar",
          "nameZH": "跳舞趴"
        },
        {
          "name": "Might of Ra",
          "symbol": "vs50mightra",
          "nameZH": "埃及神祇"
        },
        {
          "name": "Wild Hop & Drop",
          "symbol": "vs20mparty",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "狂野鹏跳青蛙"
        },
        {
          "name": "Gatot Kaca's Fury",
          "symbol": "vs20gatotfury",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "迦多铎卡之怒"
        },
        {
          "name": "Curse of the Werewolf Megaways",
          "symbol": "vswayswerewolf",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "狼人的诅咒Megaways"
        },
        {
          "name": "Triple Tigers",
          "symbol": "vs1tigers",
          "nameZH": "三只老虎"
        },
        {
          "name": "Super X",
          "symbol": "vs20superx",
          "nameZH": "超级X"
        },
        {
          "name": "Running Sushi",
          "symbol": "vswayscashconv",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "奔跑寿司"
        },
        {
          "name": "Juicy Fruits",
          "symbol": "vs50juicyfr",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "多汁水果"
        },
        {
          "name": "Sky Bounty",
          "symbol": "vs50jucier",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "海怪的高额赏金"
        },
        {
          "name": "Fortune Hit'n Roll",
          "symbol": "vs40wildrun",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "财富打带跑"
        },
        {
          "name": "Pyramid King",
          "symbol": "vs25pyramid",
          "nameZH": "王者金字塔"
        },
        {
          "name": "Wild Beach Party",
          "symbol": "vs20bchprty",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "疯狂海滩派对"
        },
        {
          "name": "Mahjong Panda",
          "symbol": "vs1024mahjpanda",
          "tags": [
            "TUMBLING"
          ],
          "nameZH": "熊猫麻将"
        },
        {
          "name": "Aztec Treasure Hunt",
          "symbol": "vs20trswild3",
          "hidePPLogo": true,
          "nameZH": "阿兹特克秘宝狩猎"
        },
        {
          "name": "Release the Kraken 2",
          "symbol": "vs20kraken2",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "解放大海怪2"
        },
        {
          "name": "Jewel Rush",
          "symbol": "vs20jewelparty",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "珍宝粉碎"
        },
        {
          "name": "Money Mouse",
          "symbol": "vs25mmouse",
          "nameZH": "招财鼠"
        },
        {
          "name": "Odds On Winner",
          "symbol": "vs10frontrun",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "领跑者加油"
        },
        {
          "name": "Drago - Jewels of Fortune",
          "symbol": "vs1600drago",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "群龙保珠-宝石致富"
        },
        {
          "name": "Wheel O'Gold",
          "symbol": "vs20multiup",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "黄金之轮"
        },
        {
          "name": "Greedy Wolf",
          "symbol": "vs20wolfie",
          "nameZH": "野狼与小猪"
        },
        {
          "name": "Wildies",
          "symbol": "vs25wildies",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "甜蜜派对"
        },
        {
          "name": "Year Of The Dragon King",
          "symbol": "vs20yotdk",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "龙王之年"
        },
        {
          "name": "Spellbinding Mystery",
          "symbol": "vs20splmystery",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "魔法的奥秘"
        },
        {
          "name": "Bermuda Riches",
          "symbol": "vs20bermuda",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "约翰猎人与百慕大财富"
        },
        {
          "name": "Blade & Fangs",
          "symbol": "vs20mergedwndw",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "刀光 & 獠牙"
        },
        {
          "name": "Drill That Gold",
          "symbol": "vs20drtgold",
          "nameZH": "钻取金子"
        },
        {
          "name": "Amazing Money Machine",
          "symbol": "vs10amm",
          "nameZH": "奇幻无比金钱取款机"
        },
        {
          "name": "Empty the Bank",
          "symbol": "vs20emptybank",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "盗空银行"
        },
        {
          "name": "Badge Blitz",
          "symbol": "vs25badge",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "徽章闪电"
        },
        {
          "name": "Chilli Heat Megaways",
          "symbol": "vswayschilheat",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "超级辣椒 Megaways"
        },
        {
          "name": "Money Stacks",
          "symbol": "vs20bblitz",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "金银闪电"
        },
        {
          "name": "Wild Depths",
          "symbol": "vs40wanderw",
          "nameZH": "深海动物"
        },
        {
          "name": "Sweet Powernudge",
          "symbol": "vs20clspwrndg",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "甜品派对 Powernudge"
        },
        {
          "name": "Book Of Tut Respin",
          "symbol": "vs10tut",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "约翰猎人与埃及之书重转"
        },
        {
          "name": "Piggy Bank Bills",
          "symbol": "vs9piggybank",
          "nameZH": "百万$小猪"
        },
        {
          "name": "Fortune Dragon",
          "symbol": "vs243fdragon",
          "nameZH": "神龙运财"
        },
        {
          "name": "Irish Charms",
          "symbol": "cs3irishcharms",
          "nameZH": "爱尔兰的魅力"
        },
        {
          "name": "5 Lions Dance",
          "symbol": "vs1024lionsd",
          "nameZH": "五狮团拜"
        },
        {
          "name": "Dragon Bonus Baccarat",
          "symbol": "bnadvanced"
        },
        {
          "name": "Crank it Up",
          "symbol": "vs20crankit",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "全面启动"
        },
        {
          "name": "Greek Gods",
          "symbol": "vs243fortseren",
          "nameZH": "希腊众神"
        },
        {
          "name": "Shield Of Sparta",
          "symbol": "vs20sparta",
          "nameZH": "斯巴达之盾"
        },
        {
          "name": "Hokkaido Wolf",
          "symbol": "vs576hokkwolf"
        },
        {
          "name": "Fire 88",
          "symbol": "vs7fire88",
          "nameZH": "88火 "
        },
        {
          "name": "Rise of Giza PowerNudge",
          "symbol": "vs10nudgeit",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "吉萨金字塔 PowerNudge"
        },
        {
          "name": "Heroic Spins",
          "symbol": "vs20shootstars",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "英雄旋转"
        },
        {
          "name": "Big Juan",
          "symbol": "vs40bigjuan",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "墨西哥餐厅"
        },
        {
          "name": "Beware The Deep Megaways",
          "symbol": "vswaysbewaretd",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "小心深海Megaways"
        },
        {
          "name": "Oodles of Noodles",
          "symbol": "vs10noodles",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "面面相趣"
        },
        {
          "name": "Three Star Fortune",
          "symbol": "vs10threestar",
          "nameZH": "三星报喜"
        },
        {
          "name": "Jade Butterfly",
          "symbol": "vs1024butterfly",
          "nameZH": "玉蝴蝶"
        },
        {
          "name": "Castle of Fire",
          "symbol": "vswaysexpandng",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "烈火城堡"
        },
        {
          "name": "Wolf Gold",
          "symbol": "vs25wolfgold",
          "nameZH": "野狼黄金"
        },
        {
          "name": "Book of Tut Megaways",
          "symbol": "vswaystut",
          "tags": [
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "约翰亨特之图特之书 Megaways"
        },
        {
          "name": "Rainbow Reels",
          "symbol": "vs40rainbowr",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "彩虹转轴"
        },
        {
          "name": "The Ultimate 5",
          "symbol": "vs20ultim5",
          "nameZH": "终极之5"
        },
        {
          "name": "Bubble Pop",
          "symbol": "vs10bblpop",
          "nameZH": "泡泡糖"
        },
        {
          "name": "Aztec Blaze",
          "symbol": "vs25kfruit",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "阿茲特克火焰"
        },
        {
          "name": "Mystic Chief",
          "symbol": "vswayswest",
          "nameZH": "神秘的酋长"
        },
        {
          "name": "Book of Golden Sands",
          "symbol": "vswaysbook",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "埃及金沙之书"
        },
        {
          "name": "Fire Archer",
          "symbol": "vs25archer",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "热火弓箭手"
        },
        {
          "name": "Lucky, Grace & Charm",
          "symbol": "vs10luckcharm",
          "nameZH": "幸运，恩典和魅力"
        },
        {
          "name": "Gears of Horus",
          "symbol": "vs20clustext",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "荷鲁斯的齿轮"
        },
        {
          "name": "Chicken Drop",
          "symbol": "vs20chickdrop",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "金鸡下蛋"
        },
        {
          "name": "Mystery Mice",
          "symbol": "vs20powerwild",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "神鼠"
        },
        {
          "name": "Medusa's Stone",
          "symbol": "vs20medusast",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "美杜莎之石"
        },
        {
          "name": "Joker's Jewel Dice",
          "symbol": "vs5jokerdice",
          "nameZH": "小丑珠宝骰子"
        },
        {
          "name": "Mystery of the Orient",
          "symbol": "vswaysmorient",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "东方的奥妙"
        },
        {
          "name": "Queen of Gods",
          "symbol": "vs10egrich"
        },
        {
          "name": "The Great Stick-up",
          "symbol": "vs20stickysymbol",
          "nameZH": "城市大盗"
        },
        {
          "name": "Starlight Princess Pachi",
          "symbol": "vswaysjapan",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "星光公主帕奇"
        },
        {
          "name": "Peking Luck",
          "symbol": "vs25peking",
          "nameZH": "好运北京"
        },
        {
          "name": "Pirates Pub",
          "symbol": "vs9outlaw",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "海盗酒吧"
        },
        {
          "name": "Little Gem Hold and Spin",
          "symbol": "vs5littlegem",
          "nameZH": "优质宝石"
        },
        {
          "name": "Cash Chips",
          "symbol": "vs20maskgame",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "现金筹码"
        },
        {
          "name": "Lucky New Year - Tiger Treasures",
          "symbol": "vs25tigeryear",
          "nameZH": "虎虎生财"
        },
        {
          "name": "Gravity Bonanza",
          "symbol": "vs20gravity",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "重力鸿运"
        },
        {
          "name": "Madame Destiny",
          "symbol": "vs10madame",
          "nameZH": "命运女士"
        },
        {
          "name": "Moonshot",
          "symbol": "vs1024moonsh",
          "tags": [
            "BUY_FEATURE"
          ],
          "hidePPLogo": true,
          "nameZH": "登陆月球"
        },
        {
          "name": "Strawberry Cocktail",
          "symbol": "vs10strawberry",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "浆果鸡尾酒"
        },
        {
          "name": "7 Piggies",
          "symbol": "vs7pigs",
          "nameZH": "7只小猪"
        },
        {
          "name": "Book Of Kingdoms",
          "symbol": "vs25bkofkngdm",
          "nameZH": "圣书帝国"
        },
        {
          "name": "Safari King",
          "symbol": "vs50safariking",
          "nameZH": "狩猎之王"
        },
        {
          "name": "Sword of Ares",
          "symbol": "vs20swordofares",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "希腊战神之剑"
        },
        {
          "name": "Pinup Girls",
          "symbol": "vs20ltng",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "海报女郎"
        },
        {
          "name": "Hot to Burn",
          "symbol": "vs5hotburn",
          "hidePPLogo": true,
          "nameZH": "热火燃烧"
        },
        {
          "name": "Ripe Rewards",
          "symbol": "vs40stckwldlvl",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": " 甜蜜奖赏"
        },
        {
          "name": "Joker's Jewels Wild",
          "symbol": "vs5jjwild",
          "nameZH": "百搭小丑珠宝"
        },
        {
          "name": "Golden Ox",
          "symbol": "vs25gldox",
          "nameZH": "鸿运你最牛"
        },
        {
          "name": "Gold Rush",
          "symbol": "vs25goldrush",
          "nameZH": "淘金热"
        },
        {
          "name": "Eye of Cleopatra",
          "symbol": "vs40cleoeye",
          "nameZH": "埃及艳后之眼"
        },
        {
          "name": "Candy Stars",
          "symbol": "vswaysstrwild",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "糖果星星"
        },
        {
          "name": "Gates of Valhalla",
          "symbol": "vs10runes",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "瓦尔哈拉之门"
        },
        {
          "name": "Asgard",
          "symbol": "vs25asgard",
          "nameZH": "仙宫"
        },
        {
          "name": "Great Lagoon",
          "symbol": "vs25lagoon",
          "nameZH": "大泻湖"
        },
        {
          "name": "Honey Honey Honey",
          "symbol": "vs20honey",
          "nameZH": "甜蜜蜜"
        },
        {
          "name": "Congo Cash XL",
          "symbol": "vswayscongcash",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "刚果财富 XL"
        },
        {
          "name": "3 Genie Wishes",
          "symbol": "vs50aladdin",
          "nameZH": "3个精灵愿望"
        },
        {
          "name": "Baccarat",
          "symbol": "bca"
        },
        {
          "name": "Dwarf & Dragon",
          "symbol": "vswaysspltsym",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "矮人和龙"
        },
        {
          "name": "Pirate Golden Age",
          "symbol": "vs20mtreasure",
          "nameZH": "海盗黄金时代"
        },
        {
          "name": "Street Racer",
          "symbol": "vs40streetracer",
          "nameZH": "街头赛车"
        },
        {
          "name": "8 Golden Dragon Challenge",
          "symbol": "vs10gdchalleng",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "8金龙挑战"
        },
        {
          "name": "Ice Lobster",
          "symbol": "vs20stickypos",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "急冻龙虾"
        },
        {
          "name": "The Alter Ego",
          "symbol": "vswaysalterego",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "双面特务"
        },
        {
          "name": "Hot to Burn 7 Deadly Free Spins",
          "symbol": "vs10hottb7fs",
          "hidePPLogo": true,
          "nameZH": "热火燃烧 – 大力七旋转"
        },
        {
          "name": "Nile Fortune",
          "symbol": "vs20nilefort",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "尼罗河财富"
        },
        {
          "name": "Golden Pig",
          "symbol": "vs25goldpig",
          "nameZH": "招财福猪"
        },
        {
          "name": "Moleionaire",
          "symbol": "vs20clreacts",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "百万富鼠"
        },
        {
          "name": "Aztec Powernudge",
          "symbol": "vs20sbpnudge",
          "tags": [
            "PAY_ANYWHERE",
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "阿兹特克 Powernudge"
        },
        {
          "name": "Diamonds are Forever 3 Lines",
          "symbol": "cs3w",
          "nameZH": "永恒钻石3线"
        },
        {
          "name": "Down the Rails",
          "symbol": "vs20underground",
          "nameZH": "伦敦地铁"
        },
        {
          "name": "Bomb Bonanza",
          "symbol": "vs25bomb",
          "tags": [
            "TUMBLING"
          ],
          "nameZH": "轰炸富矿"
        },
        {
          "name": "Star Bounty",
          "symbol": "vswayshive",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "星际赏金"
        },
        {
          "name": "Heist for the Golden Nuggets",
          "symbol": "vs20hstgldngt",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "金块大劫案"
        },
        {
          "name": "Magic Money Maze",
          "symbol": "vs10mmm",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "魔幻钱迷宫"
        },
        {
          "name": "The Magic Cauldron",
          "symbol": "vs20magicpot",
          "tags": [
            "TUMBLING",
            "CLUSTER"
          ],
          "nameZH": "魔法大釜绝佳药剂"
        },
        {
          "name": "Book of Tut",
          "symbol": "vs10bookoftut",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "约翰·亨特之埃及图特书"
        },
        {
          "name": "Fat Panda",
          "symbol": "vs20beefed",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "胖胖熊猫"
        },
        {
          "name": "Frozen Tropics",
          "symbol": "vswaysftropics",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "冰冻热带"
        },
        {
          "name": "Master Chen's Fortune",
          "symbol": "vs9chen",
          "nameZH": "陈师傅的财富"
        },
        {
          "name": "Emerald King Rainbow Road",
          "symbol": "vs20ekingrr",
          "nameZH": "翡翠之王彩虹道路"
        },
        {
          "name": "Cash Box",
          "symbol": "vs20cashmachine",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "金库宝藏"
        },
        {
          "name": "Rise of Pyramids",
          "symbol": "vswayshexhaus",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "金字塔崛起"
        },
        {
          "name": "Goblin Heist Powernudge",
          "symbol": "vs20gobnudge",
          "tags": [
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "地精时代"
        },
        {
          "name": "Big Burger Load it up with Xtra Cheese",
          "symbol": "vs10bburger",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "汉堡巨无霸"
        },
        {
          "name": "Day of Dead",
          "symbol": "vs20daydead",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "亡灵节"
        },
        {
          "name": "Vegas Nights",
          "symbol": "vs25vegas",
          "nameZH": "维加斯之夜"
        },
        {
          "name": "Egyptian Fortunes",
          "symbol": "vs20egypttrs",
          "nameZH": "埃及宿命"
        },
        {
          "name": "Gold Train",
          "symbol": "vs3train",
          "nameZH": "黄金列车"
        },
        {
          "name": "Juicy Fruits Multihold",
          "symbol": "vs50jfmulthold",
          "tags": [
            "BUY_FEATURE"
          ],
          "hidePPLogo": true,
          "nameZH": "多汁水果多方持控"
        },
        {
          "name": "Leprechaun Song",
          "symbol": "vs20leprechaun",
          "nameZH": "小妖之歌"
        },
        {
          "name": "Diamonds of Egypt",
          "symbol": "vswayseternity",
          "nameZH": "埃及钻石"
        },
        {
          "name": "Hot Chilli",
          "symbol": "vs9hotroll",
          "nameZH": "超级辣"
        },
        {
          "name": "3 Kingdoms - Battle of Red Cliffs",
          "symbol": "vs25kingdoms",
          "nameZH": "三国"
        },
        {
          "name": "Colossal Cash Zone",
          "symbol": "vs20colcashzone",
          "nameZH": "巨大的现金区域"
        },
        {
          "name": "Temujin Treasures",
          "symbol": "vs1024temuj",
          "tags": [
            "BUY_FEATURE"
          ],
          "hidePPLogo": true,
          "nameZH": "铁木真宝物"
        },
        {
          "name": "Lucky New Year",
          "symbol": "vs25newyear",
          "nameZH": "幸运新年"
        },
        {
          "name": "Forging Wilds",
          "symbol": "vs20forgewilds",
          "nameZH": "狂野锻造"
        },
        {
          "name": "Pub Kings",
          "symbol": "vs20lvlup",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "酒吧之王"
        },
        {
          "name": "Devilicious",
          "symbol": "vs20devilic",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "超级恶魔"
        },
        {
          "name": "Hot To Burn Multiplier",
          "symbol": "vs5hotbmult",
          "hidePPLogo": true,
          "nameZH": "热火燃烧暴增"
        },
        {
          "name": "Gemstone",
          "symbol": "vs5gemstone",
          "nameZH": "宝石 "
        },
        {
          "name": "Gem Elevator",
          "symbol": "vs20elevclust",
          "tags": [
            "TUMBLING",
            "CLUSTER"
          ],
          "nameZH": "宝石天梯"
        },
        {
          "name": "Dynamite Diggin Doug",
          "symbol": "vs10dyndigd",
          "tags": [
            "BUY_FEATURE"
          ],
          "hidePPLogo": true,
          "nameZH": "劲爆矿工德哥"
        },
        {
          "name": "Mysterious Egypt",
          "symbol": "vs10wildtut",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "古埃及神话"
        },
        {
          "name": "Cyclops Smash",
          "symbol": "vs20earthquake",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "独眼巨人猛击"
        },
        {
          "name": "Fishin Reels",
          "symbol": "vs10goldfish",
          "nameZH": "大鱼奇缘"
        },
        {
          "name": "Mighty Munching Melons",
          "symbol": "vs20mmmelon",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "大啖甜瓜"
        },
        {
          "name": "John Hunter And The Mayan Gods",
          "symbol": "vs10mayangods",
          "nameZH": "约翰猎人与玛雅神迹"
        },
        {
          "name": "Pirate Gold Deluxe",
          "symbol": "vs40pirgold",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "黄金海盗加强版"
        },
        {
          "name": "Vegas Magic",
          "symbol": "vs20vegasmagic",
          "tags": [
            "TUMBLING"
          ],
          "nameZH": "魔力维加斯"
        },
        {
          "name": "Book of Aztec King",
          "symbol": "vs10bookazteck",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "阿兹特克国王之书"
        },
        {
          "name": "Trees of Treasure",
          "symbol": "vs20treesot",
          "nameZH": "招财树"
        },
        {
          "name": "Secret City Gold",
          "symbol": "vs25spgldways",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "神秘黄金之城"
        },
        {
          "name": "Piggy Bankers",
          "symbol": "vs20piggybank",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "小猪银行家"
        },
        {
          "name": "Starz Megaways",
          "symbol": "vs117649starz",
          "tags": [
            "MEGAWAYS"
          ]
        },
        {
          "name": "Pot of Fortune",
          "symbol": "vs20stckwldsc",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "幸运之壶"
        },
        {
          "name": "The Wild Machine",
          "symbol": "vs40madwheel",
          "nameZH": "疯狂机器"
        },
        {
          "name": "Leprechaun Carol",
          "symbol": "vs20leprexmas",
          "nameZH": "小妖之歌圣诞版"
        },
        {
          "name": "Hockey Attack",
          "symbol": "vs88hockattack",
          "tags": [
            "TUMBLING"
          ],
          "nameZH": "冰球竞赛"
        },
        {
          "name": "Crown of Fire",
          "symbol": "vs10crownfire",
          "nameZH": "王冠之火"
        },
        {
          "name": "Rainbow Gold",
          "symbol": "vs20rainbowg",
          "nameZH": "黄金彩虹"
        },
        {
          "name": "Loki's Riches",
          "symbol": "vs20loksriches",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "洛基财富"
        },
        {
          "name": "Towering Fortunes",
          "symbol": "vs20theights",
          "nameZH": "摩天大楼"
        },
        {
          "name": "Voodoo Magic",
          "symbol": "vs40voodoo",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "恶毒魔法"
        },
        {
          "name": "Sea Fantasy",
          "symbol": "vswaysseastory",
          "nameZH": "海洋幻想"
        },
        {
          "name": "Lamp Of Infinity",
          "symbol": "vs20lampinf",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "无限神灯"
        },
        {
          "name": "Wild Pixies",
          "symbol": "vs20wildpix",
          "nameZH": "野精灵"
        },
        {
          "name": "Hellvis Wild",
          "symbol": "vs243nudge4gold",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "野蛮恶魔"
        },
        {
          "name": "Wild Walker",
          "symbol": "vs25walker",
          "nameZH": "行尸走肉"
        },
        {
          "name": "Knight Hot Spotz",
          "symbol": "vs25spotz",
          "nameZH": "骑士热点"
        },
        {
          "name": "Rise of Samurai",
          "symbol": "vs25samurai",
          "nameZH": "崛起的武士"
        },
        {
          "name": "Diamond Cascade",
          "symbol": "vs50dmdcascade",
          "tags": [
            "TUMBLING",
            "CLUSTER",
            "BUY_FEATURE"
          ],
          "nameZH": "钻石瀑布"
        },
        {
          "name": "Hercules and Pegasus",
          "symbol": "vs20hercpeg",
          "nameZH": "大力神和飞马"
        },
        {
          "name": "Vampires vs Wolves",
          "symbol": "vs10vampwolf",
          "nameZH": "吸血鬼vs狼"
        },
        {
          "name": "Wild Wild Bananas",
          "symbol": "vswayswwhex",
          "nameZH": "狂野美味香蕉 "
        },
        {
          "name": "Dragon Kingdom",
          "symbol": "vs25dragonkingdom",
          "nameZH": "龙之国度"
        },
        {
          "name": "Gold Oasis",
          "symbol": "vswaysincwnd",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "黄金绿洲"
        },
        {
          "name": "Pixie Wings",
          "symbol": "vs50pixie",
          "nameZH": "精灵翅膀"
        },
        {
          "name": "Smugglers Cove",
          "symbol": "vs20smugcove",
          "nameZH": "走私者海湾"
        },
        {
          "name": "The Great Chicken Escape",
          "symbol": "vs20chicken",
          "nameZH": "小鸡大逃亡"
        },
        {
          "name": "Dragon Kingdom - Eyes of Fire",
          "symbol": "vs5drmystery",
          "nameZH": "神龙宝藏雄火之眼"
        },
        {
          "name": "Lucky Dragons",
          "symbol": "vs50chinesecharms",
          "nameZH": "飞龙在天"
        },
        {
          "name": "Fairytale Fortune",
          "symbol": "vs15fairytale",
          "nameZH": "童话财富"
        },
        {
          "name": "Old Gold Miner Megaways",
          "symbol": "vswaysoldminer",
          "tags": [
            "TUMBLING",
            "MEGAWAYS",
            "BUY_FEATURE"
          ],
          "nameZH": "黄金老矿工Megaways"
        },
        {
          "name": "Disco Lady",
          "symbol": "vs243discolady",
          "nameZH": "迪斯科女士"
        },
        {
          "name": "Striking Hot 5",
          "symbol": "vs5strh",
          "nameZH": "火热闪烁5"
        },
        {
          "name": "Jane Hunter and the Mask of Montezuma",
          "symbol": "vs10jnmntzma",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "珍亨特与蒙特苏马的面具"
        },
        {
          "name": "Journey to the West",
          "symbol": "vs25journey",
          "nameZH": "西游记"
        },
        {
          "name": "Tundra's Fortune",
          "symbol": "vswaysraghex",
          "nameZH": "极地财富"
        },
        {
          "name": "Aladdin and the Sorcerer",
          "symbol": "vs20aladdinsorc"
        },
        {
          "name": "Ding Dong Christmas Bells",
          "symbol": "vs10ddcbells",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "圣诞响叮当"
        },
        {
          "name": "Frogs & Bugs",
          "symbol": "vswaysfrbugs",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "蛙群家族"
        },
        {
          "name": "Demon Pots",
          "symbol": "vs40demonpots",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "魔法之壶"
        },
        {
          "name": "North Guardians",
          "symbol": "vs50northgard",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "北境守护者"
        },
        {
          "name": "Treasure Horse",
          "symbol": "vs18mashang",
          "nameZH": "马上有钱"
        },
        {
          "name": "7 Monkeys",
          "symbol": "vs7monkeys",
          "nameZH": "7只猴子"
        },
        {
          "name": "Fu Fu Fu",
          "symbol": "vs1fufufu",
          "nameZH": "福福福"
        },
        {
          "name": "Santa",
          "symbol": "vs20santa",
          "nameZH": "圣诞老人"
        },
        {
          "name": "Jackpot Hunter",
          "symbol": "vs20jhunter",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "奖金猎人"
        },
        {
          "name": "Snakes and Ladders Megadice",
          "symbol": "vs10snakeladd",
          "nameZH": "蛇梯棋超级骰子"
        },
        {
          "name": "Kingdom of the Dead",
          "symbol": "vs10kingofdth",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "亡者王国"
        },
        {
          "name": "Mysterious",
          "symbol": "vs4096mystery",
          "nameZH": "神秘之城"
        },
        {
          "name": "Koi Pond",
          "symbol": "vs243koipond",
          "nameZH": "招财锦鲤"
        },
        {
          "name": "Emerald King",
          "symbol": "vs20eking",
          "nameZH": "翡翠之王"
        },
        {
          "name": "The Knight King",
          "symbol": "vs20sknights",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "骑士之王"
        },
        {
          "name": "Mighty Kong",
          "symbol": "vs50kingkong",
          "nameZH": "无敌金刚"
        },
        {
          "name": "Lobster Bob's Sea Food and Win It",
          "symbol": "vs20lobseafd",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "鲍勃龙虾海鲜大餐"
        },
        {
          "name": "The Tiger Warrior",
          "symbol": "vs25tigerwar",
          "nameZH": "武松打虎"
        },
        {
          "name": "Fire Hot 100",
          "symbol": "vs100firehot",
          "nameZH": "热火 100"
        },
        {
          "name": "Wild Booster",
          "symbol": "vs20wildboost",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "百搭加强器"
        },
        {
          "name": "Great Reef",
          "symbol": "vs25sea"
        },
        {
          "name": "Da Vinci's Treasure",
          "symbol": "vs25davinci",
          "nameZH": "达芬奇宝藏"
        },
        {
          "name": "Aztec Treasure",
          "symbol": "vs7776secrets",
          "tags": [
            "MEGAWAYS"
          ],
          "nameZH": "阿兹特克秘宝"
        },
        {
          "name": "Busy Bees",
          "symbol": "vs20bl"
        },
        {
          "name": "Snakes & Ladders - Snake Eyes",
          "symbol": "vs10snakeeyes",
          "nameZH": "蛇和梯子：蛇眼"
        },
        {
          "name": "Wild Gladiator",
          "symbol": "vs25gladiator",
          "nameZH": "狂野角斗士"
        },
        {
          "name": "African Elephant",
          "symbol": "vs20hotzone",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "非洲大象"
        },
        {
          "name": "Jungle Gorilla",
          "symbol": "vs20gorilla",
          "nameZH": "深林之猩"
        },
        {
          "name": "Cheeky Emperor",
          "symbol": "vs243ckemp",
          "nameZH": "调皮的皇帝"
        },
        {
          "name": "Devil's 13",
          "symbol": "vs13g",
          "nameZH": "恶魔13"
        },
        {
          "name": "Gems of Serengeti",
          "symbol": "vs20lcount",
          "nameZH": "塞伦盖蒂之宝石"
        },
        {
          "name": "Jasmine Dreams",
          "symbol": "vs20mvwild",
          "nameZH": "阿拉伯之梦"
        },
        {
          "name": "Fruit Rainbow",
          "symbol": "vs40frrainbow",
          "nameZH": "彩虹水果"
        },
        {
          "name": "Timber Stacks",
          "symbol": "vswaystimber",
          "tags": [
            "TUMBLING",
            "BUY_FEATURE"
          ],
          "nameZH": "森林大冒险"
        },
        {
          "name": "Hercules Son of Zeus",
          "symbol": "vs50hercules",
          "nameZH": "宙斯之子赫拉克勒斯"
        },
        {
          "name": "Shining Hot 20",
          "symbol": "vs20sh",
          "nameZH": "閃亮 热火 20"
        },
        {
          "name": "The Red Queen",
          "symbol": "vswaysredqueen",
          "nameZH": "红桃王后"
        },
        {
          "name": "Good Luck & Good Fortune",
          "symbol": "vs10luckfort",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "鸿运当头"
        },
        {
          "name": "Star Pirates Code",
          "symbol": "vs10starpirate",
          "nameZH": "星际海盗代码"
        },
        {
          "name": "Chase For Glory",
          "symbol": "vswayscfglory",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "追逐荣光"
        },
        {
          "name": "Hot Safari",
          "symbol": "vs25safari",
          "nameZH": "野生动物园"
        },
        {
          "name": "Triple Dragons",
          "symbol": "vs5trdragons",
          "nameZH": "龙龙龙"
        },
        {
          "name": "Chicken Chase",
          "symbol": "vs10chkchase",
          "nameZH": "农场追逐"
        },
        {
          "name": "Cash Bonanza",
          "symbol": "vswaysbankbonz",
          "nameZH": "水果银行"
        },
        {
          "name": "Fire Hot 5",
          "symbol": "vs5firehot",
          "nameZH": "热火 5"
        },
        {
          "name": "Shining Hot 40",
          "symbol": "vs40sh",
          "nameZH": "热火 40"
        },
        {
          "name": "Shining Hot 100",
          "symbol": "vs100sh",
          "nameZH": "热火 100"
        },
        {
          "name": "Magic Journey",
          "symbol": "vs8magicjourn",
          "nameZH": "西游争霸"
        },
        {
          "name": "Lobster Bob's Crazy Crab Shack",
          "symbol": "vs20lobcrab",
          "tags": [
            "BUY_FEATURE"
          ],
          "nameZH": "鲍勃龙虾的疯狂螃蟹屋"
        },
        {
          "name": "Lady Godiva",
          "symbol": "vs20godiva"
        },
        {
          "name": "Hot To Burn Extreme",
          "symbol": "vs40hotburnx",
          "tags": [
            "BUY_FEATURE"
          ],
          "hidePPLogo": true,
          "nameZH": "极度热火燃烧"
        },
        {
          "name": "Fire Hot 40",
          "symbol": "vs40firehot",
          "nameZH": "热火 40"
        },
        {
          "name": "Dwarven Gold Deluxe",
          "symbol": "vs25dwarves_new"
        },
        {
          "name": "Fire Hot 20",
          "symbol": "vs20fh",
          "nameZH": "热火 20"
        },
        {
          "name": "Shining Hot 5",
          "symbol": "vs5sh",
          "nameZH": "火热闪烁5"
        }
      ],
      activePromo: {},
      lastPlayed: [0, 1, 2, 3, 4, 5, 6, 7, 8],
      landingPage: [9, 3, 10, 11, 12, 13, 14, 15, 16, 17, 0, 18, 19, 20],
      vendorLobby: {
        new: {
          games: [
            16, 21, 11, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35,
            36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
          ],
        },
        all: {
          games: [
            9, 3, 10, 11, 12, 13, 14, 15, 16, 17, 0, 18, 19, 20, 46, 47, 48, 29,
            21, 49, 50, 51, 31, 52, 53, 54, 55, 56, 57, 58, 22, 59, 60, 61, 27,
            62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 24, 73, 74, 75, 76, 77,
            78, 79, 80, 81, 82, 35, 83, 84, 85, 86, 87, 1, 88, 89, 90, 91, 92,
            38, 93, 25, 94, 95, 96, 8, 97, 98, 99, 37, 100, 101, 102, 103, 104,
            105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117,
            118, 119, 120, 121, 122, 26, 123, 124, 28, 125, 126, 127, 128, 129,
            130, 131, 132, 133, 134, 135, 39, 136, 137, 138, 139, 140, 141, 142,
            143, 144, 145, 32, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155,
            156, 157, 158, 159, 160, 161, 162, 163, 164, 165, 166, 167, 168,
            169, 170, 171, 172, 173, 174, 175, 176, 177, 178, 179, 180, 181,
            182, 183, 184, 185, 186, 23, 187, 188, 189, 190, 191, 192, 33, 2,
            193, 194, 45, 195, 196, 197, 198, 199, 200, 34, 201, 202, 203, 204,
            205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217,
            218, 6, 42, 36, 219, 220, 221, 222, 223, 224, 225, 226, 227, 228,
            229, 230, 231, 232, 233, 234, 235, 236, 237, 238, 239, 240, 241,
            242, 243, 244, 245, 246, 247, 248, 249, 250, 251, 252, 253, 254,
            255, 256, 257, 258, 259, 260, 261, 262, 263, 264, 265, 266, 267,
            268, 269, 270, 271, 272, 273, 274, 275, 5, 276, 277, 278, 279, 280,
            281, 282, 283, 284, 285, 286, 287, 288, 289, 290, 291, 40, 292, 293,
            4, 294, 295, 296, 297, 298, 299, 300, 301, 302, 303, 304, 305, 306,
            307, 308, 309, 310, 311, 312, 313, 314, 315, 316, 43, 317, 318, 319,
            320, 321, 322, 323, 324, 325, 326, 327, 328, 329, 330, 331, 332,
            333, 334, 335, 336, 337, 338, 44, 339, 340, 341, 342, 343, 344, 345,
            346, 347, 348, 349, 350, 351, 352, 353, 354, 355, 356, 357, 358,
            359, 360, 361, 362, 363, 364, 365, 366, 367, 368, 369, 370, 371,
            372, 373, 374, 375, 376, 377, 378, 379, 380, 381, 382, 383, 384,
            385, 386, 387, 388, 389, 390, 391, 392, 393, 394, 395, 396, 397,
            398, 399, 400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410,
            411, 412, 413, 414, 415, 416, 417, 418, 419, 420, 421, 422, 423,
            424, 7, 425, 426, 427, 428, 429, 430, 431, 432, 433, 434, 435, 436,
            437, 438, 439, 440, 441, 442, 443, 444, 445, 446, 447, 448, 449,
            450, 451, 452, 453, 454, 455, 456, 457, 458, 459, 460, 461, 462,
            463, 464, 465, 466, 467, 468, 469, 470, 471, 472, 473, 474, 475,
            476, 477, 478, 479, 480, 481, 482, 483, 484, 485, 486, 487, 488,
            489, 490, 30, 41, 491,
          ],
        },
        hot: {
          games: [
            20, 17, 46, 50, 49, 57, 14, 63, 52, 16, 59, 68, 55, 82, 10, 69, 24,
            76, 90, 21, 30, 65, 27, 67, 73, 81, 94, 61,
          ],
        },
      },
      playerSections: { lastPlayed: [0, 1, 2, 3, 4, 5, 6, 7, 8] },
      vendor: "SLOTS",
    },
  ],
  error: 0,
  description: "OK",
};

games.data[0].vendorConfig.gameLaunchURL =
  games.data[0].vendorConfig.gameLaunchURL.replaceAll(
    "{{server_url}}",
    baseConfig.server_url
  );
games.data[0].vendorConfig.gameIconsURL =
  games.data[0].vendorConfig.gameIconsURL.replaceAll(
    "{{static_url}}",
    baseConfig.static_url
  );

const testUserId = "1";

export { baseConfig, games, testUserId };
