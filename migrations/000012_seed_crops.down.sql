-- Faqat seed profillari va ishlatilmagan seed ekinlari o'chadi
-- (solution_batches / seasons RESTRICT).
DELETE FROM crop_profiles
WHERE owner_user_id IS NULL
  AND source = 'Sonneveld & Voogt 2009; Sonneveld & Straver 1994'
  AND crop_id IN (SELECT id FROM crops WHERE code IN ('tomato', 'cucumber'))
  AND NOT EXISTS (SELECT 1 FROM solution_batches b WHERE b.crop_profile_id = crop_profiles.id);

DELETE FROM crops c
WHERE c.code IN ('tomato', 'cucumber', 'pepper', 'strawberry', 'lettuce')
  AND NOT EXISTS (SELECT 1 FROM crop_profiles p WHERE p.crop_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM seasons s WHERE s.crop_id = c.id);
