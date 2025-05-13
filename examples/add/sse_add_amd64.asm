use64
; add.asm: Adds two float64 values from an Args struct and stores the result.
; Input: RAX = pointer to Args struct { In: [2]float64, Out: float64 }
;        - In[0] at [RAX+0]  (float64, 8 bytes)
;        - In[1] at [RAX+8]  (float64, 8 bytes)
;        - Out   at [RAX+16] (float64, 8 bytes)
; Output: Stores sum of In[0] + In[1] in Out
movsd xmm0, [rax]      ; Load In[0] into XMM0
addsd xmm0, [rax+8]    ; Add In[1] to XMM0
movsd [rax+16], xmm0   ; Store result in Out
ret                    ; Return