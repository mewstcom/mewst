# typed: false
# frozen_string_literal: true

RSpec.describe "GET /email_confirmations/new", type: :request do
  it "email_confirmation_idがsessionにないとき、ルートパスにリダイレクトすること" do
    get "/email_confirmations/new"

    expect(response).to redirect_to(root_path)
  end

  it "すでにサインイン済みでもemail_confirmation_idがsessionにないとき、ルートパスにリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/email_confirmations/new"

    expect(response).to redirect_to(root_path)
  end

  it "email_confirmation_idがsessionにあるとき、正常にレスポンスが返されること" do
    email_confirmation = FactoryBot.create(:email_confirmation_record)
    set_session(email_confirmation_id: email_confirmation.id)

    get "/email_confirmations/new"

    expect(response).to have_http_status(:ok)
  end

  it "サインイン済みでemail_confirmation_idがsessionにあるとき、正常にレスポンスが返されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)
    email_confirmation = FactoryBot.create(:email_confirmation_record)
    set_session(email_confirmation_id: email_confirmation.id)

    get "/email_confirmations/new"

    expect(response).to have_http_status(:ok)
  end
end
