# typed: false
# frozen_string_literal: true

RSpec.describe "GET /public", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    get "/public"

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、パブリックタイムラインが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/public"

    expect(response).to have_http_status(:ok)
  end

  it "ログインしているとき、投稿フォームが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/public"

    expect(response.body).to include("post_form[content]")
  end

  it "ログインしているとき、公開された投稿が表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)
    FactoryBot.create(:post_record, content: "最初の投稿", published_at: 2.hours.ago)
    FactoryBot.create(:post_record, content: "2番目の投稿", published_at: 1.hour.ago)

    get "/public"

    expect(response.body).to include("最初の投稿")
    expect(response.body).to include("2番目の投稿")
  end

  it "ログインしているとき、投稿が新しい順に表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)
    FactoryBot.create(:post_record, content: "古い投稿", published_at: 2.hours.ago)
    FactoryBot.create(:post_record, content: "新しい投稿", published_at: 1.hour.ago)

    get "/public"

    # 新しい投稿が古い投稿より先に表示されることを確認
    expect(response.body.index("新しい投稿")).to be < response.body.index("古い投稿")
  end

  it "ログインしているとき、削除された投稿は表示されないこと" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)
    FactoryBot.create(:post_record, content: "表示される投稿")
    FactoryBot.create(:post_record, content: "削除された投稿", discarded_at: Time.current)

    get "/public"

    expect(response.body).to include("表示される投稿")
    expect(response.body).not_to include("削除された投稿")
  end

  it "ログインしているとき、ページネーションが機能すること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)
    # 20件の投稿を作成
    20.times do |i|
      FactoryBot.create(:post_record, content: "ページネーション投稿#{i}", published_at: i.hours.ago)
    end

    # 最初のページを取得（最大15件）
    get "/public"
    expect(response).to have_http_status(:ok)

    # 最初のページには15件の投稿が表示される
    post_count_first_page = response.body.scan(/ページネーション投稿\d+/).uniq.count
    expect(post_count_first_page).to eq(15)
  end

  it "ログインしているとき、無効なafterパラメータで301リダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/public", params: {after: "invalid_cursor"}

    expect(response).to redirect_to("/public")
    expect(response).to have_http_status(:moved_permanently)
  end

  it "ログインしているとき、無効なbeforeパラメータで301リダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/public", params: {before: "invalid_cursor"}

    expect(response).to redirect_to("/public")
    expect(response).to have_http_status(:moved_permanently)
  end

  it "ログインしているとき、最大15件の投稿が表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)
    20.times do |i|
      FactoryBot.create(:post_record, content: "投稿#{i}", published_at: i.minutes.ago)
    end

    get "/public"

    # 最大15件の投稿が表示されることを確認
    post_count = response.body.scan(/投稿\d+/).uniq.count
    expect(post_count).to eq(15)
  end
end
