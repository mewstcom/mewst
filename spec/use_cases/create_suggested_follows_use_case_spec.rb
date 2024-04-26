# typed: false
# frozen_string_literal: true

RSpec.describe CreateSuggestedFollowsUseCase do
  context "おすすめするプロフィールが存在するとき" do
    def setup_data
      source_profile = FactoryBot.create(:profile)
      profile_1 = FactoryBot.create(:profile)
      profile_2 = FactoryBot.create(:profile)
      profile_3 = FactoryBot.create(:profile)

      FollowProfileUseCase.new.call(profile: source_profile, target_profile: profile_1)
      FollowProfileUseCase.new.call(profile: profile_1, target_profile: profile_2)
      FollowProfileUseCase.new.call(profile: profile_2, target_profile: profile_3)

      {source_profile:, profile_1:, profile_2:, profile_3:}
    end

    it "おすすめプロフィールを作成すること" do
      setup_data => {source_profile:, profile_1:, profile_2:, profile_3:}

      expect(source_profile.followees).to contain_exactly(profile_1)
      expect(profile_1.followees).to contain_exactly(profile_2)
      expect(profile_2.followees).to contain_exactly(profile_3)
      expect(source_profile.suggested_follows).to be_empty

      CreateSuggestedFollowsUseCase.new.call(source_profile:)

      # 自分がフォローしている人 (profile_1) がフォローしている人 (profile_2) と
      # その人がフォローしている人 (profile_3) をおすすめするレコードが作成されているはず
      expect(source_profile.suggested_followees).to contain_exactly(profile_2, profile_3)
    end
  end
end
