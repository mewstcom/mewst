# typed: false
# frozen_string_literal: true

RSpec.describe Buttons::FollowButtonComponent do
  describe "System spec", type: :system do
    it do
      visit "/rails/view_components/buttons/follow_button_component/default"

      expect(page).to have_content "Success!!!"
    end
  end
end
