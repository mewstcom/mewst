# typed: false
# frozen_string_literal: true

RSpec.describe "POST /posts", type: :request do
  before do
    FactoryBot.create(:oauth_application, :mewst_web)
  end

  it "認証されていないユーザーの場合、ルートパスにリダイレクトすること" do
    post "/posts", params: {
      post_form: {
        content: "テスト投稿",
        canonical_url: "",
        with_frame: false
      }
    }

    expect(response).to redirect_to(root_path)
  end

  it "認証されたユーザーで有効な投稿データの場合、投稿が作成されてホームにリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    expect(PostRecord.count).to eq(0)

    post "/posts", params: {
      post_form: {
        content: "テスト投稿",
        canonical_url: "",
        with_frame: false
      }
    }

    expect(response).to redirect_to(home_path)
    expect(flash[:notice]).to eq("投稿しました")
    expect(PostRecord.count).to eq(1)

    created_post = PostRecord.first
    expect(created_post.content).to eq("テスト投稿")
    expect(created_post.profile_record).to eq(actor.profile_record)
  end

  it "認証されたユーザーでwith_frameがtrueの場合、投稿が作成されてTurbo Streamで応答すること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    expect(PostRecord.count).to eq(0)

    post "/posts", params: {
      post_form: {
        content: "テスト投稿",
        canonical_url: "",
        with_frame: true
      }
    }

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("text/vnd.turbo-stream.html")
    expect(PostRecord.count).to eq(1)

    created_post = PostRecord.first
    expect(created_post.content).to eq("テスト投稿")
    expect(created_post.profile_record).to eq(actor.profile_record)
  end

  it "認証されたユーザーで既存のLinkのcanonical_urlが設定された場合、投稿が作成されてLinkが関連付けられること" do
    actor = FactoryBot.create(:actor)
    link = FactoryBot.create(:link_record, canonical_url: "https://example.com")
    sign_in(actor)

    expect(PostRecord.count).to eq(0)

    post "/posts", params: {
      post_form: {
        content: "テスト投稿",
        canonical_url: "https://example.com",
        with_frame: false
      }
    }

    expect(response).to redirect_to(home_path)
    expect(PostRecord.count).to eq(1)

    created_post = PostRecord.first
    expect(created_post.content).to eq("テスト投稿")
    expect(created_post.link_record).to eq(link)
    expect(created_post.link_record.canonical_url).to eq("https://example.com")
  end

  it "認証されたユーザーで投稿内容が空の場合、422エラーが返ること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    post "/posts", params: {
      post_form: {
        content: "",
        canonical_url: "",
        with_frame: false
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(PostRecord.count).to eq(0)
  end

  it "認証されたユーザーで投稿内容が最大文字数を超える場合、422エラーが返ること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    long_content = "a" * (PostRecord::MAXIMUM_CONTENT_LENGTH + 1)

    post "/posts", params: {
      post_form: {
        content: long_content,
        canonical_url: "",
        with_frame: false
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(PostRecord.count).to eq(0)
  end

  it "認証されたユーザーで無効なcanonical_urlが設定された場合、422エラーが返ること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    post "/posts", params: {
      post_form: {
        content: "テスト投稿",
        canonical_url: "invalid-url",
        with_frame: false
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(PostRecord.count).to eq(0)
  end
end
