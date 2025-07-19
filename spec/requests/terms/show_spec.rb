# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "GET /terms", type: :request do
  it "利用規約のURLにリダイレクトすること" do
    terms_url = "https://example.com/terms"
    allow(Rails.configuration.mewst).to receive(:[]).with("terms_url").and_return(terms_url)

    get "/terms"

    expect(response).to redirect_to(terms_url)
  end

  it "外部ホストへのリダイレクトを許可すること" do
    terms_url = "https://external-domain.com/terms"
    allow(Rails.configuration.mewst).to receive(:[]).with("terms_url").and_return(terms_url)

    get "/terms"

    expect(response).to redirect_to(terms_url)
  end
end
