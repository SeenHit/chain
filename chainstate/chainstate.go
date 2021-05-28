package chainstate

import (
	"fmt"
	"github.com/hacash/chain/hashtreedb"
	"github.com/hacash/chain/tinykvdb"
	"github.com/hacash/core/fields"
	"github.com/hacash/core/interfaces"
	"github.com/hacash/core/stores"
	"path"
)

type ChainState struct {
	//temporaryDataDir string

	// parent state
	base *ChainState

	// config
	config *ChainStateConfig

	// status
	laststatusDB *tinykvdb.TinyKVDB

	// state
	balanceDB *hashtreedb.HashTreeDB
	diamondDB *hashtreedb.HashTreeDB
	channelDB *hashtreedb.HashTreeDB
	movebtcDB *hashtreedb.HashTreeDB // 转移BTC记录
	lockblsDB *hashtreedb.HashTreeDB // 线性锁仓地址 key len = 18
	dmdlendDB *hashtreedb.HashTreeDB // 钻石系统借贷 key len = 14

	// store
	datastore          interfaces.BlockStore
	datastore_mycreate interfaces.BlockStore

	// data hold
	pendingBlockHeight *uint64
	pendingBlockHash   fields.Hash

	submitStoreDiamond *stores.DiamondSmelt

	prev288BlockTimestamp           uint64
	lastestBlockHeadAndMeta         interfaces.Block
	lastestDiamond                  *stores.DiamondSmelt
	lastestBlockHeadAndMeta_forSave interfaces.Block
	lastestDiamond_forSave          *stores.DiamondSmelt
	totalSupply                     *stores.TotalSupply
	totalSupply_forSave             *stores.TotalSupply

	// status
	isInTxPool bool
}

func NewChainState(cnf *ChainStateConfig) (*ChainState, error) {
	return newChainStateEx(cnf, false)
}

func newChainStateEx(cnf *ChainStateConfig, isSubBranchTemporary bool) (*ChainState, error) {
	var laststatusDB *tinykvdb.TinyKVDB = nil
	// is temp state
	//var temporaryDataDir string
	if isSubBranchTemporary {
		//randstr := strconv.FormatUint(uint64(rand.Uint32()), 10)
		//temporaryDataDir = path.Join(os.TempDir(), "hacash_state_temp_"+randstr)
		//cnf.Datadir = temporaryDataDir
	} else {
		// laststatusDB
		lsdb, lserr := tinykvdb.NewTinyKVDB(path.Join(cnf.Datadir, "laststatus"), true)
		if lserr != nil {
			fmt.Println("tinykvdb.NewTinyKVDB Error", lserr)
			return nil, lserr
		}
		laststatusDB = lsdb
	}
	// balanceDB
	// fmt.Println("balanceDB dir:", path.Join(cnf.Datadir, "balance"))
	// os.MkdirAll(path.Join(cnf.Datadir, "balance"), os.ModePerm)
	blscnf := hashtreedb.NewHashTreeDBConfig(path.Join(cnf.Datadir, "balance"), stores.BalanceSize, 21)
	//blscnf.KeyReverse = true
	if isSubBranchTemporary {
		blscnf.MemoryStorage = true // 内存数据库
		//blscnf.KeepDeleteMark = true
		//blscnf.SaveMarkBeforeValue = true
	} else {
		blscnf.LevelDB = true // 使用 level db
	}
	balanceDB := hashtreedb.NewHashTreeDB(blscnf)

	/////////////  TEST BEGIN /////////////
	//go func() {
	//	if !isSubBranchTemporary {
	//		time.Sleep(time.Second * 5)
	//		Test_print_all_address_balance(balanceDB)
	//	}
	//}()
	/////////////  TEST END   /////////////

	// diamondDB
	dmdcnf := hashtreedb.NewHashTreeDBConfig(path.Join(cnf.Datadir, "diamond"), stores.DiamondSize, 6)
	//dmdcnf.KeyPrefixSupplement = 10
	if isSubBranchTemporary {
		dmdcnf.MemoryStorage = true // 内存数据库
		// dmdcnf.ForbidGC = true
		//dmdcnf.KeepDeleteMark = true
		//dmdcnf.SaveMarkBeforeValue = true
	} else {
		dmdcnf.LevelDB = true // 使用 level db
		//dmdcnf.FileDividePartitionLevel = 1
	}
	diamondDB := hashtreedb.NewHashTreeDB(dmdcnf)
	// channelDB
	chlcnf := hashtreedb.NewHashTreeDBConfig(path.Join(cnf.Datadir, "channel"), stores.ChannelSize, 16)
	if isSubBranchTemporary {
		chlcnf.MemoryStorage = true // 内存数据库
		//chlcnf.ForbidGC = true
		//chlcnf.KeepDeleteMark = true
		//chlcnf.SaveMarkBeforeValue = true
	} else {
		chlcnf.LevelDB = true // 使用 level db
		//chlcnf.FileDividePartitionLevel = 1
	}
	channelDB := hashtreedb.NewHashTreeDB(chlcnf)
	// movebtcDB
	mvbtcnf := hashtreedb.NewHashTreeDBConfig(path.Join(cnf.Datadir, "movebtc"), 32, 4)
	//mvbtcnf.KeyPrefixSupplement = 4
	if isSubBranchTemporary {
		mvbtcnf.MemoryStorage = true // 内存数据库
		//mvbtcnf.ForbidGC = true
		//mvbtcnf.KeepDeleteMark = true
		//mvbtcnf.SaveMarkBeforeValue = true
	} else {
		mvbtcnf.LevelDB = true // 使用 level db
		//mvbtcnf.FileDividePartitionLevel = 1
	}
	movebtcDB := hashtreedb.NewHashTreeDB(mvbtcnf)
	// lockblsDB
	lkblscnf := hashtreedb.NewHashTreeDBConfig(path.Join(cnf.Datadir, "lockbls"), stores.LockblsSize, stores.LockblsIdLength)
	//blscnf.KeyReverse = true // 倒排key
	if isSubBranchTemporary {
		lkblscnf.MemoryStorage = true // 内存数据库
		//lkblscnf.ForbidGC = true
		//lkblscnf.KeepDeleteMark = true
		//lkblscnf.SaveMarkBeforeValue = true
	} else {
		lkblscnf.LevelDB = true // 使用 level db
		//lkblscnf.FileDividePartitionLevel = 1
	}
	lockblsDB := hashtreedb.NewHashTreeDB(lkblscnf)
	// lockblsDB
	dmdlendcnf := hashtreedb.NewHashTreeDBConfig(path.Join(cnf.Datadir, "dmdlend"), 0, stores.DiamondLendingIdLength)
	if isSubBranchTemporary {
		dmdlendcnf.MemoryStorage = true // 内存数据库
	} else {
		dmdlendcnf.LevelDB = true // 使用 level db
	}
	dmdlendDB := hashtreedb.NewHashTreeDB(dmdlendcnf)
	// return ok
	cs := &ChainState{
		config: cnf,
		// temporaryDataDir:      temporaryDataDir,
		base:                  nil,
		laststatusDB:          laststatusDB,
		balanceDB:             balanceDB,
		diamondDB:             diamondDB,
		channelDB:             channelDB,
		movebtcDB:             movebtcDB,
		lockblsDB:             lockblsDB,
		dmdlendDB:             dmdlendDB,
		prev288BlockTimestamp: 0,
		pendingBlockHeight:    nil,
		pendingBlockHash:      nil,
		isInTxPool:            false,
	}
	return cs, nil
}

// interface api
func (cs *ChainState) Destory() {
	cs.DestoryTemporary()
}

func (cs *ChainState) Close() {
	if cs.laststatusDB != nil {
		cs.laststatusDB.Close()
	}
	if cs.balanceDB != nil {
		cs.balanceDB.Close()
	}
	if cs.channelDB != nil {
		cs.channelDB.Close()
	}
	if cs.diamondDB != nil {
		cs.diamondDB.Close()
	}
	if cs.movebtcDB != nil {
		cs.movebtcDB.Close()
	}
	if cs.lockblsDB != nil {
		cs.lockblsDB.Close()
	}
	if cs.dmdlendDB != nil {
		cs.dmdlendDB.Close()
	}
	// close mycreate store
	if cs.datastore_mycreate != nil {
		cs.datastore_mycreate.Close()
	}
}

// destory temporary
func (cs *ChainState) DestoryTemporary() {
	//cs.Close()
	/*
		// remove temp data dir
		e1 := os.RemoveAll(cs.temporaryDataDir)
		if e1 != nil {
			fmt.Println(e1)
		}
	*/
}

// chain data store
func (cs *ChainState) BlockStore() interfaces.BlockStore {
	if cs.datastore != nil {
		return cs.datastore
	}
	if cs.base != nil {
		cs.datastore = cs.base.BlockStore() // copy
		return cs.datastore
	}
	return cs.datastore
}

func (cs *ChainState) SetBlockStore(store interfaces.BlockStore) error {
	if cs.base != nil {
		return fmt.Errorf("Can only be set chainstore in the final state.")
	}
	cs.datastore = store
	cs.datastore_mycreate = store // 我创建的
	return nil
}

// merge write
func (cs *ChainState) MergeCoverWriteChainState(src *ChainState) error {
	// copy status
	if src.pendingBlockHeight != nil {
		cs.SetPendingBlockHeight(*src.pendingBlockHeight)
	}
	if src.pendingBlockHash != nil {
		cs.SetPendingBlockHash(src.pendingBlockHash)
	}
	if src.submitStoreDiamond != nil {
		e := cs.SetPendingSubmitStoreDiamond(src.submitStoreDiamond)
		if e != nil {
			return e
		}
	}
	if src.lastestBlockHeadAndMeta != nil {
		e := cs.SetLastestBlockHeadAndMeta(src.lastestBlockHeadAndMeta)
		if e != nil {
			return e
		}
	}
	if src.lastestDiamond != nil {
		e := cs.SetLastestDiamond(src.lastestDiamond)
		if e != nil {
			return e
		}
	}
	if src.totalSupply != nil {
		e := cs.UpdateSetTotalSupply(src.totalSupply)
		if e != nil {
			return e
		}
	}

	//  COPY COVER WRITE STATE

	e1 := cs.balanceDB.TraversalCopy(src.balanceDB)
	if e1 != nil {
		return e1
	}
	e2 := cs.diamondDB.TraversalCopy(src.diamondDB)
	if e2 != nil {
		return e2
	}
	e3 := cs.channelDB.TraversalCopy(src.channelDB)
	if e3 != nil {
		return e3
	}
	e5 := cs.movebtcDB.TraversalCopy(src.movebtcDB)
	if e5 != nil {
		return e5
	}
	e6 := cs.lockblsDB.TraversalCopy(src.lockblsDB)
	if e6 != nil {
		return e6
	}
	e7 := cs.dmdlendDB.TraversalCopy(src.dmdlendDB)
	if e7 != nil {
		return e7
	}

	// copy ok

	return nil
}

// interface api
func (cs *ChainState) Fork() (interfaces.ChainState, error) {
	return cs.NewSubBranchTemporaryChainState()
}

func (cs *ChainState) IsDatabaseVersionRebuildMode() bool {
	if cs.base != nil {
		// 递归向上
		return cs.base.IsDatabaseVersionRebuildMode()
	}
	// 返回最终配置
	return cs.config.DatabaseVersionRebuildMode
}

// fork sub
func (cs *ChainState) NewSubBranchTemporaryChainState() (*ChainState, error) {

	tempcnf := NewEmptyChainStateConfig()
	// 拷贝一些配置
	tempcnf.BTCMoveCheckEnable = cs.config.BTCMoveCheckEnable
	tempcnf.BTCMoveCheckLogsURL = cs.config.BTCMoveCheckLogsURL
	// 拷贝完毕
	newTempState, err1 := newChainStateEx(tempcnf, true)
	if err1 != nil {
		return nil, err1
	}
	// set base
	newTempState.base = cs
	// ok
	return newTempState, nil
}

// submit to write disk
func (cs *ChainState) SubmitDataStoreWriteToInvariableDisk(block interfaces.Block) error {
	if cs.base != nil {
		panic("Can only be saved in the final state.")
	}
	if cs.pendingBlockHash == nil {
		//return fmt.Errorf("Block pending hash not set.")
		panic("pending block hash not be set.")
	}
	//
	store := cs.BlockStore()
	if store == nil {
		return fmt.Errorf("Not find BlockStore object.")
	}
	// save status
	e0 := cs.SetLastestBlockHeadAndMeta(block)
	if e0 != nil {
		return e0
	}
	e1 := cs.IncompleteSaveLastestBlockHeadAndMeta()
	if e1 != nil {
		return e1
	}
	e2 := cs.IncompleteSaveLastestDiamond()
	if e2 != nil {
		return e2
	}
	e9 := cs.IncompleteSaveTotalSupply()
	if e9 != nil {
		return e9
	}
	// save diamond
	if cs.submitStoreDiamond != nil {
		cs.submitStoreDiamond.ContainBlockHash = cs.pendingBlockHash // copy
		e := store.SaveDiamond(cs.submitStoreDiamond)
		if e != nil {
			return e
		}
	}
	// save block data
	e3 := store.SaveBlockUniteTransactions(block)
	if e3 != nil {
		return e3
	}
	// reset clear
	cs.pendingBlockHash = nil
	cs.pendingBlockHeight = nil
	cs.submitStoreDiamond = nil
	// ok
	return nil
}

////////////////////////////////////////////////////

func Test_print_all_address_balance(db *hashtreedb.HashTreeDB) {

	total_address_count := int64(0)

	total_hac := float64(0)
	total_btc := int64(0)
	total_hacd := int(0)

	iter := db.LevelDB.NewIterator(nil, nil)
	for iter.Next() {
		//fmt.Printf("key:%s, value:%s\n", iter.Key(), iter.Value())
		addr := fields.Address(iter.Key())
		var bls = stores.Balance{}
		bls.Parse(iter.Value(), 0)
		hacfltn := bls.Hacash.ToMei()
		if hacfltn == 0 && bls.Satoshi == 0 && bls.Diamond == 0 {
			continue
		}
		fmt.Printf("% 4d %-34s % 12.4f % 6d %d\n", total_address_count+1, addr.ToReadable(), hacfltn, bls.Diamond, bls.Satoshi)
		total_hac += float64(hacfltn)
		total_btc += int64(bls.Satoshi)
		total_hacd += int(bls.Diamond)
		total_address_count++
	}
	iter.Release()

	fmt.Println("------------------\n[TOTAL]", total_address_count, "address, HAC:", total_hac, "SAT:", total_btc, "HACD:", total_hacd)
}
