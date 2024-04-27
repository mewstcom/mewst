# typed: false
# frozen_string_literal: true

RSpec.describe "トップページ", type: :system do
  it do
    visit "/"

    expect(page).to have_content "現在ベータ版です"
  end
end
