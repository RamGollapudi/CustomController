package main

import "fmt"

func main()  {
	fmt.Println(`hello`)
	sample:=[]int{2,5,1,3,4,10}
	//target:=3
	//fmt.Println(sum(sample,target))
	fmt.Println(secondLowest(sample))
}
func sum( i []int,target int) (int,int) {

	for ind,_ :=range i{
		for j:=ind+1;j<=len(i)-1;j++{
			if i[ind]+i[j]==target{
				return ind,j
			}
		}
	}
	return 0,0
}

func secondLowest(nums []int)int{
	for i:=len(nums);i>0;i--{
		for j:=1;j<i;j++{
			if nums[j-1]>nums[j]{
				swap:=nums[j]
				nums[j]=nums[j-1]
				nums[j-1]=swap
			}
		}
	}
return nums[1]
}