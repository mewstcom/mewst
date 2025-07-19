# typed: false
# frozen_string_literal: true

RSpec.describe "GET /community", type: :request do
  it "コミュニティURLにリダイレクトすること" do
    community_url = "https://example.com/community"
    allow(Rails.configuration.mewst).to receive(:[]).and_call_original
    allow(Rails.configuration.mewst).to receive(:[]).with("community_url").and_return(community_url)

    get "/community"

    expect(response).to have_http_status(:redirect)
    expect(response).to redirect_to(community_url)
  end

  it "ログインしているとき、コミュニティURLにリダイレクトすること" do
    community_url = "https://example.com/community"
    allow(Rails.configuration.mewst).to receive(:[]).and_call_original
    allow(Rails.configuration.mewst).to receive(:[]).with("community_url").and_return(community_url)
    actor = FactoryBot.create(:actor)
    sign_in actor

    get "/community"

    expect(response).to have_http_status(:redirect)
    expect(response).to redirect_to(community_url)
  end
end
