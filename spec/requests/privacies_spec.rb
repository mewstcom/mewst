# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "GET /privacy", type: :request do
  it "プライバシーポリシーのURLにリダイレクトすること" do
    privacy_url = "https://example.com/privacy"
    allow(Rails.configuration.mewst).to receive(:[]).with("privacy_url").and_return(privacy_url)

    get "/privacy"

    expect(response).to redirect_to(privacy_url)
  end

  it "外部ホストへのリダイレクトを許可すること" do
    privacy_url = "https://external-domain.com/privacy"
    allow(Rails.configuration.mewst).to receive(:[]).with("privacy_url").and_return(privacy_url)

    get "/privacy"

    expect(response).to redirect_to(privacy_url)
  end
end
