# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /settings/user", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    patch "/settings/user", params: {
      user_form: {
        locale: "ja",
        time_zone: "Asia/Tokyo"
      }
    }

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、有効なパラメータでユーザー設定が更新されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/user", params: {
      user_form: {
        locale: "ja",
        time_zone: "Asia/Tokyo"
      }
    }

    expect(response).to redirect_to("/settings/user")
    expect(flash[:notice]).to eq("ユーザー設定を更新しました。")

    user = actor.user.reload
    expect(user.locale).to eq("ja")
    expect(user.time_zone).to eq("Asia/Tokyo")
  end

  it "ログインしているとき、英語ロケールとニューヨークのタイムゾーンで設定が更新されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/user", params: {
      user_form: {
        locale: "en",
        time_zone: "America/New_York"
      }
    }

    expect(response).to redirect_to("/settings/user")

    user = actor.user.reload
    expect(user.locale).to eq("en")
    expect(user.time_zone).to eq("America/New_York")
  end

  it "ログインしているとき、無効なロケールが指定されたらエラーページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/user", params: {
      user_form: {
        locale: "invalid",
        time_zone: "Asia/Tokyo"
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("言語は一覧にありません")
  end

  it "ログインしているとき、無効なタイムゾーンが指定されたらエラーページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/user", params: {
      user_form: {
        locale: "ja",
        time_zone: "Invalid/TimeZone"
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("タイムゾーンは一覧にありません")
  end

  it "ログインしているとき、ロケールが空の場合エラーページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/user", params: {
      user_form: {
        locale: "",
        time_zone: "Asia/Tokyo"
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("言語を入力してください")
  end

  it "ログインしているとき、タイムゾーンが空の場合エラーページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/user", params: {
      user_form: {
        locale: "ja",
        time_zone: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("タイムゾーンを入力してください")
  end

  it "ログインしているとき、user_formパラメータが存在しない場合エラーになること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/user", params: {}

    expect(response).to have_http_status(:bad_request)
  end
end
