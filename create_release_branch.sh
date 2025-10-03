#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
# This is crucial for scripts like this, so a cherry-pick failure
# (like a merge conflict) stops the script.
set -e

commits=(
    "9b20eddacded43dba432f932774b38188571bbf8" #PR 484
    "b566efd3bc7ace39bb8400a1430a5652376cd7d3" #PR 481
    "c58bc1790278eb5b034e9a9e2f7b03bd2ff6bfc3" #PR 482
    "18861f29e95bb4f57f33975ebfd47495f5ec4c12" #PR 485
    # "c2b7a246ac51360c7b78f7c996553a56f98dccef" #PR 483
    # "fdf3dba7c9f36889ab46d075f4dc9328e07099ab" #PR 488 revert of 483
    "d16eeafafaab22647bec3da9413673ad5708633d" #PR 489
    "d77d2cf35e825ea7c21d84b0178ee474306b3855" #PR 493
    "96aba1bd8afb8d5e5828177b900181ece0dd3806" #PR 490
    "627ff5bab03189c6f66dd249e814e264017295a7" #PR 494
    "c7f31161cb4a4a845d7fbc812b0733120ba65c8a" #PR 491
    "56fb3a843bda7731b66e43defd3f38d626bf86a3" #PR 495
    "a0df3a6bba6314653d2accedec960d01f135f162" #PR 499
    "0466233df013aedb949961fb9219afefe07cef32" #PR 501
    "d4517ed6262f00e216eb5be0e3731214f7bcf191" #PR 502
    "70a545d63225c3c2c02f69df6ca0626a9a060f18" #PR 497
    "4090a955380cd415b33a9f77415746bb04c77a67" #PR 505
    "1dc547782ebcd1f612a7894a0db1d74888b3b1ad" #PR 508
    "848c1213afd0678626d2cfed44c005bdb9c08882" #PR 507
    "7096c3b17b57ffaa6272b480ea3b201d0f3f896c" #PR 511
    "4d214bd4a88c81b3fb4f37ee3bb0c80d00e52fc9" #PR 512
    # "90355e334cd7d36b7b1b9cacfb0623d92a883d4f" #PR 510
    "bb2f990318c579a53f362dff9a2fff17ac178070" # Luis version of PR 510
    "37d6cabf25605fbee7ff4bf26c0fac631bc21366" #PR 513
    "6222e78335e9a6a20e503e8c732f6c7f1ffbe517" #PR 515
    #"22c47992e6fc7d47532aabdc948d5d95e43f1f35" #PR 514
    "f770355dc58ea6ee8ea8dce7d45b905517263deb" # Luis version of PR 514
    "d170851413a1b7fc3c8baf9090f1e2e9e70e842a" #PR 516
    "d37b0489fc918dc66b67854c4aae864b55938a14" #PR 517
    "a5b3f5c11694ff1df7af57ca92fa3ae302123b3c" #PR 518
    "d0b2617e5f6aef9102d9d0f2a65268db1742ea17" #PR 521
    #"22e159900dc1e16094f227e76dbbd474d913fc19" #PR 524
    "e5b223944808943956eb464543458f0a20c2f45f" # Luis version of PR 524
    "d5eb064e3d56a1adfc22323d0000bdc857ab669a" #PR 519
    "81c7f4a5a32818c2bf1bc0e568ed45b1f4659cd8" #PR 527
    "b696c09e68bb9c60853fd3bcb1bdf005e8a0d8c2" #PR 525
    "e677d8071a936d18fe77ff82caa0902a1866eec1" #PR 523
    "7dd8a6fe6876e76beb039373ff809cadb8544a69" #PR 531
    "54a8e5438b899e1b7fc0b48682f404419c504dda" #PR 528
    "5bc202e86997f07d182fa9799ba7a54e809c2f80" #PR 522
    "36ceb44d15ea4def8f9b32a8ba66b96bf698c21a" #PR 526
    "54bcf484f60cb6b777e83a3bfbd3dc8f1ff9ea60" #PR 530
    "c58d29407aa6bea9d10127fbd7e600ade6b3ff01" #PR 536
    "d7dea26d3d574fde0e349be74e3c1c14f4653a4d" #PR 537
    "c20052a1b780569f7675f338bde5d41536a6b87a" #PR 534
    "e58e77e6f33ce76cb03316a0751372795364fa0c" #PR 535
    "ee032b92d056ac4f75657f95b8956a354822e412" #PR 539
    "d27136e14368cf6fabbfb4dcdc32ba6034dbb8e3" #PR 538
)

echo "Setting up release candidate branch..."
git fetch
git checkout v2.1
git checkout -b release-v2.1-candidate
echo -e "Branch 'release-v2.1-candidate' created from 'v2.1'.\n"

echo "Cherry-picking commits..."
for commit in "${commits[@]}"; do
    echo "Picking commit: $commit"
    git cherry-pick -x "$commit"
done
echo -e "All commits have been cherry-picked.\n"

echo "Running make to ensure build integrity..."
make
echo -e "Build successful.\n"

echo "Comparing to luis/gas_subsidies_towards_2.1.2 branch..."
git diff release-v2.1-candidate..origin/luis/gas_subsidies_towards_2.1.2
echo -e "Comparison complete.\n"

echo "3 commits were altered between the main and release branches."
echo -e "\nDiff between PR 510 versions:"
git diff 90355e334cd7d36b7b1b9cacfb0623d92a883d4f..bb2f990318c579a53f362dff9a2fff17ac178070
echo -e "\nDiff between PR 514 versions:"
git diff 22c47992e6fc7d47532aabdc948d5d95e43f1f35..f770355dc58ea6ee8ea8dce7d45b905517263deb
echo -e "\nDiff between PR 524 versions:"
git diff 22e159900dc1e16094f227e76dbbd474d913fc19..e5b223944808943956eb464543458f0a20c2f45f
echo -e "\nPlease review the differences above.\n"