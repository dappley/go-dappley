// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//
package core

import "github.com/libp2p/go-libp2p-peer"

type Consensus interface{
	Validate(block *Block) bool
	Start()
	Stop()
	StartNewBlockMinting()
	Setup(NetService, string)
	SetTargetBit(int)
}

type NetService interface{
	SendBlock(block *Block) error
	GetPeerID() peer.ID
	GetBlockchain() *Blockchain
}

